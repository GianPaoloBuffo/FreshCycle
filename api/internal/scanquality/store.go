package scanquality

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db          *pgxpool.Pool
	environment string
}

func NewPostgresStore(db *pgxpool.Pool, environment string) PostgresStore {
	return PostgresStore{
		db:          db,
		environment: NormalizeEnvironment(environment),
	}
}

func (s PostgresStore) RecordScan(ctx context.Context, input ScanRecordInput) (ScanRecordResult, error) {
	if s.db == nil {
		return ScanRecordResult{}, errors.New("scan quality store is not configured")
	}

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return ScanRecordResult{}, errors.New("scan quality user id is required")
	}

	environment := NormalizeEnvironment(firstNonEmpty(input.Environment, s.environment))
	outcome := input.Outcome
	if outcome == "" {
		outcome = OutcomeSuccess
	}

	imageHash := firstNonEmpty(input.ScanResult.ImageHash, ImageHash(input.ScanInput))
	ocrText := scanOCRText(input.ScanInput, input.ScanResult)
	ocrTextHash := TextHash(ocrText)
	fieldConfidences := mustJSON(scanFieldConfidences(input.ScanResult))
	symbolClasses := SymbolClasses(input.ScanResult.SymbolDetections)
	fullPhotoStored := false
	imageSize := nullableInt(len(input.ScanInput.Content))
	confidence := nullableFloat(input.ScanResult.Confidence)
	captureQualityScore := input.Request.CaptureQualityScore
	imageScope := firstNonEmpty(input.Request.ImageScope, "label_crop")

	var scanEventID string
	err := s.db.QueryRow(ctx, `
		insert into public.scan_events (
			user_id,
			environment,
			outcome,
			error_code,
			capture_source,
			image_scope,
			image_hash,
			image_mime_type,
			image_size_bytes,
			full_photo_stored,
			has_client_ocr,
			ocr_text_hash,
			client_symbol_count,
			capture_quality_score,
			capture_quality_issues,
			provider,
			route,
			cache_hit,
			paid_fallback_used,
			fallback_calls_avoided,
			confidence,
			needs_user_confirmation,
			uncertain_fields,
			routing_reasons,
			symbol_classes,
			field_confidences,
			retention_expires_at
		)
		values (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26::jsonb,
			coalesce(
				(
					select timezone('utc', now()) + image_hash_retention
					from public.scan_retention_policies
					where environment = $2
				),
				timezone('utc', now()) + interval '180 days'
			)
		)
		returning id
	`,
		userID,
		environment,
		outcome,
		emptyToNil(input.ErrorCode),
		emptyToNil(input.Request.CaptureSource),
		imageScope,
		emptyToNil(imageHash),
		emptyToNil(input.ScanInput.MIMEType),
		imageSize,
		fullPhotoStored,
		input.ScanInput.ClientOCR != nil,
		emptyToNil(ocrTextHash),
		len(input.ScanInput.ClientSymbols),
		captureQualityScore,
		normalizedStrings(input.Request.CaptureQualityHints),
		emptyToNil(input.ScanResult.Provider),
		emptyToNil(input.ScanResult.Route),
		input.ScanResult.CacheHit,
		input.ScanResult.PaidFallbackUsed,
		input.ScanResult.FallbackCallsAvoided,
		confidence,
		input.ScanResult.NeedsUserConfirmation,
		normalizedStrings(input.ScanResult.UncertainFields),
		normalizedStrings(input.ScanResult.RoutingReasons),
		symbolClasses,
		fieldConfidences,
	).Scan(&scanEventID)
	if err != nil {
		return ScanRecordResult{}, fmt.Errorf("insert scan event: %w", err)
	}

	reasons := ReviewReasons(input.ScanResult, input.Request)
	priority := ActiveLearningPriority(input.ScanResult, reasons)
	if outcome == OutcomeFailure {
		reasons = appendUnique(reasons, "scan_failed")
		if input.ErrorCode != "" {
			reasons = appendUnique(reasons, "scan_failed:"+input.ErrorCode)
		}
		if priority < 0.80 {
			priority = 0.80
		}
	}
	if len(reasons) == 0 {
		return ScanRecordResult{ScanEventID: scanEventID}, nil
	}

	reviewQueueID, err := s.createReviewQueue(ctx, createReviewQueueInput{
		UserID:           userID,
		ScanEventID:      scanEventID,
		Environment:      environment,
		Priority:         priority,
		ReviewReasons:    reasons,
		CroppedImagePath: input.Request.CroppedImagePath,
		ImageHash:        imageHash,
		OCROutput:        ocrText,
		DetectorOutput:   mustJSON(input.ScanResult.SymbolDetections),
		ModelResult:      mustJSON(input.ScanResult),
	})
	if err != nil {
		return ScanRecordResult{}, err
	}

	if err := s.createActiveLearningCandidate(ctx, activeLearningCandidateInput{
		UserID:          userID,
		ScanEventID:     scanEventID,
		ReviewQueueID:   reviewQueueID,
		Priority:        priority,
		Reasons:         reasons,
		Signals:         sourceSignals(input.ScanResult, reasons),
		UncommonClasses: UncommonClasses(input.ScanResult.SymbolDetections),
	}); err != nil {
		return ScanRecordResult{}, err
	}

	return ScanRecordResult{
		ScanEventID:            scanEventID,
		ReviewQueueID:          reviewQueueID,
		ReviewReasons:          reasons,
		ActiveLearningPriority: priority,
	}, nil
}

func (s PostgresStore) RecordReviewDecision(ctx context.Context, input ReviewDecisionInput) (ReviewDecisionResult, error) {
	if s.db == nil {
		return ReviewDecisionResult{}, errors.New("scan quality store is not configured")
	}

	userID := strings.TrimSpace(input.UserID)
	decision := NormalizeDecision(input.Decision)
	if userID == "" {
		return ReviewDecisionResult{}, errors.New("scan quality user id is required")
	}
	if decision == "" {
		return ReviewDecisionResult{}, errors.New("valid review decision is required")
	}
	if strings.TrimSpace(input.ScanEventID) == "" && strings.TrimSpace(input.ReviewQueueID) == "" {
		return ReviewDecisionResult{}, errors.New("scan_event_id or review_queue_id is required")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ReviewDecisionResult{}, fmt.Errorf("begin review decision: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	reviewQueueID := strings.TrimSpace(input.ReviewQueueID)
	if reviewQueueID == "" && (decision == DecisionCorrect || decision == DecisionNeedsLabel || decision == DecisionPrivacyDelete) {
		reviewQueueID, err = s.ensureReviewQueueForDecision(ctx, tx, input, decision)
		if err != nil {
			return ReviewDecisionResult{}, err
		}
	}

	var reviewDecisionID string
	if err := tx.QueryRow(ctx, `
		insert into public.scan_review_decisions (
			review_queue_id,
			scan_event_id,
			user_id,
			decision,
			corrected_fields,
			field_corrections,
			final_user_correction,
			notes
		)
		values ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8)
		returning id
	`,
		emptyToNil(reviewQueueID),
		emptyToNil(input.ScanEventID),
		userID,
		decision,
		normalizedStrings(input.CorrectedFields),
		mustJSON(input.FieldCorrections),
		mustJSON(input.FinalUserCorrection),
		emptyToNil(input.Notes),
	).Scan(&reviewDecisionID); err != nil {
		return ReviewDecisionResult{}, fmt.Errorf("insert review decision: %w", err)
	}

	if reviewQueueID != "" {
		if err := s.updateReviewQueueForDecision(ctx, tx, reviewQueueID, userID, decision, input); err != nil {
			return ReviewDecisionResult{}, err
		}
	}

	annotationExampleID := ""
	if decision == DecisionCorrect || decision == DecisionNeedsLabel {
		annotationExampleID, err = s.promoteReviewQueueToAnnotation(ctx, tx, reviewQueueID, userID)
		if err != nil {
			return ReviewDecisionResult{}, err
		}
		if err := s.markActiveLearningSelected(ctx, tx, input.ScanEventID, reviewQueueID, annotationExampleID, userID); err != nil {
			return ReviewDecisionResult{}, err
		}
	}

	if decision == DecisionPrivacyDelete {
		if err := s.privacyDeleteRelatedRecords(ctx, tx, input.ScanEventID, reviewQueueID, userID); err != nil {
			return ReviewDecisionResult{}, err
		}
	}

	for _, correction := range input.FieldCorrections {
		if !isSupportedAccuracyField(correction.FieldName) {
			continue
		}
		if err := s.insertFieldAccuracyEvent(ctx, tx, reviewDecisionID, input.ScanEventID, userID, correction); err != nil {
			return ReviewDecisionResult{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ReviewDecisionResult{}, fmt.Errorf("commit review decision: %w", err)
	}

	return ReviewDecisionResult{
		ReviewDecisionID:    reviewDecisionID,
		ReviewQueueID:       reviewQueueID,
		AnnotationExampleID: annotationExampleID,
	}, nil
}

type createReviewQueueInput struct {
	UserID           string
	ScanEventID      string
	Environment      string
	Priority         float64
	ReviewReasons    []string
	CroppedImagePath string
	ImageHash        string
	OCROutput        string
	DetectorOutput   string
	ModelResult      string
}

func (s PostgresStore) createReviewQueue(ctx context.Context, input createReviewQueueInput) (string, error) {
	var reviewQueueID string
	if err := s.db.QueryRow(ctx, `
		insert into public.scan_review_queue (
			scan_event_id,
			user_id,
			priority,
			review_reasons,
			cropped_label_image_path,
			image_hash,
			ocr_output,
			detector_output,
			model_result,
			retention_expires_at
		)
		values (
			$1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb,
			coalesce(
				(
					select timezone('utc', now()) + review_record_retention
					from public.scan_retention_policies
					where environment = $10
				),
				timezone('utc', now()) + interval '180 days'
			)
		)
		returning id
	`,
		input.ScanEventID,
		input.UserID,
		input.Priority,
		normalizedStrings(input.ReviewReasons),
		emptyToNil(input.CroppedImagePath),
		emptyToNil(input.ImageHash),
		emptyToNil(input.OCROutput),
		input.DetectorOutput,
		input.ModelResult,
		NormalizeEnvironment(input.Environment),
	).Scan(&reviewQueueID); err != nil {
		return "", fmt.Errorf("insert scan review queue: %w", err)
	}
	return reviewQueueID, nil
}

type activeLearningCandidateInput struct {
	UserID          string
	ScanEventID     string
	ReviewQueueID   string
	Priority        float64
	Reasons         []string
	Signals         []string
	UncommonClasses []string
}

func (s PostgresStore) createActiveLearningCandidate(ctx context.Context, input activeLearningCandidateInput) error {
	_, err := s.db.Exec(ctx, `
		insert into public.active_learning_candidates (
			scan_event_id,
			review_queue_id,
			user_id,
			priority_score,
			priority_reasons,
			source_signals,
			uncommon_classes,
			detector_version
		)
		values ($1, $2, $3, $4, $5, $6, $7, 'v0.1.0')
	`,
		emptyToNil(input.ScanEventID),
		emptyToNil(input.ReviewQueueID),
		input.UserID,
		input.Priority,
		normalizedStrings(input.Reasons),
		normalizedStrings(input.Signals),
		normalizedStrings(input.UncommonClasses),
	)
	if err != nil {
		return fmt.Errorf("insert active learning candidate: %w", err)
	}
	return nil
}

func (s PostgresStore) ensureReviewQueueForDecision(ctx context.Context, tx pgx.Tx, input ReviewDecisionInput, decision string) (string, error) {
	var reviewQueueID string
	err := tx.QueryRow(ctx, `
		insert into public.scan_review_queue (
			scan_event_id,
			user_id,
			status,
			priority,
			review_reasons,
			image_hash,
			final_user_correction,
			decision,
			decided_at,
			retention_expires_at
		)
		select
			events.id,
			events.user_id,
			$3,
			case when $4 = 'privacy_delete' then 1 else 0.90 end,
			array[$5]::text[],
			coalesce(nullif($6, ''), events.image_hash),
			$7::jsonb,
			$4,
			timezone('utc', now()),
			coalesce(
				(
					select timezone('utc', now()) + review_record_retention
					from public.scan_retention_policies
					where environment = events.environment
				),
				timezone('utc', now()) + interval '180 days'
			)
		from public.scan_events events
		where events.id = $1
			and events.user_id = $2
		returning id
	`,
		input.ScanEventID,
		input.UserID,
		DecisionStatus(decision),
		decision,
		"user_correction",
		input.ImageHash,
		mustJSON(input.FinalUserCorrection),
	).Scan(&reviewQueueID)
	if err != nil {
		return "", fmt.Errorf("create review queue for decision: %w", err)
	}
	return reviewQueueID, nil
}

func (s PostgresStore) updateReviewQueueForDecision(ctx context.Context, tx pgx.Tx, reviewQueueID string, userID string, decision string, input ReviewDecisionInput) error {
	if decision == DecisionPrivacyDelete {
		_, err := tx.Exec(ctx, `
			update public.scan_review_queue
			set
				status = 'privacy_deleted',
				decision = $3,
				decided_at = timezone('utc', now()),
				privacy_delete_requested_at = timezone('utc', now()),
				redacted_at = timezone('utc', now()),
				cropped_label_image_path = null,
				image_hash = null,
				ocr_output = null,
				detector_output = '{}'::jsonb,
				model_result = '{}'::jsonb,
				final_user_correction = '{}'::jsonb
			where id = $1 and user_id = $2
		`, reviewQueueID, userID, decision)
		if err != nil {
			return fmt.Errorf("privacy delete review queue: %w", err)
		}
		return nil
	}

	_, err := tx.Exec(ctx, `
		update public.scan_review_queue
		set
			status = $3,
			decision = $4,
			decided_at = timezone('utc', now()),
			final_user_correction = $5::jsonb,
			review_reasons = case
				when $4 = 'correct' then array_append(review_reasons, 'user_correction')
				when $4 = 'needs_label' then array_append(review_reasons, 'needs_label')
				else review_reasons
			end
		where id = $1 and user_id = $2
	`, reviewQueueID, userID, DecisionStatus(decision), decision, mustJSON(input.FinalUserCorrection))
	if err != nil {
		return fmt.Errorf("update review queue decision: %w", err)
	}
	return nil
}

func (s PostgresStore) promoteReviewQueueToAnnotation(ctx context.Context, tx pgx.Tx, reviewQueueID string, userID string) (string, error) {
	if reviewQueueID == "" {
		return "", nil
	}

	var existingID string
	err := tx.QueryRow(ctx, `
		select annotation_example_id
		from public.scan_review_queue
		where id = $1 and user_id = $2 and annotation_example_id is not null
	`, reviewQueueID, userID).Scan(&existingID)
	if err == nil {
		return existingID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("get existing annotation example: %w", err)
	}

	var annotationID string
	err = tx.QueryRow(ctx, `
		insert into public.scan_annotation_examples (
			review_queue_id,
			user_id,
			image_hash,
			cropped_label_image_path,
			source,
			status,
			priority,
			retention_expires_at
		)
		select
			id,
			user_id,
			image_hash,
			cropped_label_image_path,
			'review_queue',
			'candidate',
			greatest(priority, 0.75),
			coalesce(
				(
					select timezone('utc', now()) + annotation_retention
					from public.scan_retention_policies
					where environment = 'production'
				),
				timezone('utc', now()) + interval '400 days'
			)
		from public.scan_review_queue
		where id = $1 and user_id = $2
		returning id
	`, reviewQueueID, userID).Scan(&annotationID)
	if err != nil {
		return "", fmt.Errorf("promote review queue to annotation: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		update public.scan_review_queue
		set annotation_example_id = $3
		where id = $1 and user_id = $2
	`, reviewQueueID, userID, annotationID); err != nil {
		return "", fmt.Errorf("link annotation example: %w", err)
	}

	return annotationID, nil
}

func (s PostgresStore) markActiveLearningSelected(ctx context.Context, tx pgx.Tx, scanEventID string, reviewQueueID string, annotationID string, userID string) error {
	tag, err := tx.Exec(ctx, `
		update public.active_learning_candidates
		set
			status = 'selected',
			annotation_example_id = coalesce(nullif($3, '')::uuid, annotation_example_id),
			priority_score = greatest(priority_score, 0.90),
			priority_reasons = array_append(priority_reasons, 'user_correction')
		where user_id = $4
			and (
				(nullif($1, '') is not null and scan_event_id = nullif($1, '')::uuid)
				or (nullif($2, '') is not null and review_queue_id = nullif($2, '')::uuid)
			)
	`, scanEventID, reviewQueueID, annotationID, userID)
	if err != nil {
		return fmt.Errorf("update active learning candidate: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	_, err = tx.Exec(ctx, `
		insert into public.active_learning_candidates (
			scan_event_id,
			review_queue_id,
			annotation_example_id,
			user_id,
			status,
			priority_score,
			priority_reasons,
			source_signals,
			detector_version
		)
		values (
			nullif($1, '')::uuid,
			nullif($2, '')::uuid,
			nullif($3, '')::uuid,
			$4,
			'selected',
			0.90,
			array['user_correction']::text[],
			array['review_decision']::text[],
			'v0.1.0'
		)
	`, scanEventID, reviewQueueID, annotationID, userID)
	if err != nil {
		return fmt.Errorf("insert selected active learning candidate: %w", err)
	}
	return nil
}

func (s PostgresStore) privacyDeleteRelatedRecords(ctx context.Context, tx pgx.Tx, scanEventID string, reviewQueueID string, userID string) error {
	_, err := tx.Exec(ctx, `
		update public.scan_annotation_examples
		set
			status = 'privacy_deleted',
			privacy_delete_requested_at = timezone('utc', now()),
			redacted_at = timezone('utc', now()),
			image_hash = null,
			cropped_label_image_path = null,
			class_counts = '{}'::jsonb
		where user_id = $2
			and (
				nullif($1, '') is not null
				and review_queue_id = nullif($1, '')::uuid
			)
	`, reviewQueueID, userID)
	if err != nil {
		return fmt.Errorf("privacy delete annotation examples: %w", err)
	}

	_, err = tx.Exec(ctx, `
		delete from public.scan_symbol_annotations
		where annotation_example_id in (
			select id
			from public.scan_annotation_examples
			where user_id = $2
				and nullif($1, '') is not null
				and review_queue_id = nullif($1, '')::uuid
		)
	`, reviewQueueID, userID)
	if err != nil {
		return fmt.Errorf("privacy delete symbol annotations: %w", err)
	}

	_, err = tx.Exec(ctx, `
		update public.active_learning_candidates
		set status = 'privacy_deleted'
		where user_id = $3
			and (
				(nullif($1, '') is not null and scan_event_id = nullif($1, '')::uuid)
				or (nullif($2, '') is not null and review_queue_id = nullif($2, '')::uuid)
			)
	`, scanEventID, reviewQueueID, userID)
	if err != nil {
		return fmt.Errorf("privacy delete active learning candidates: %w", err)
	}
	return nil
}

func (s PostgresStore) insertFieldAccuracyEvent(ctx context.Context, tx pgx.Tx, reviewDecisionID string, scanEventID string, userID string, correction FieldCorrection) error {
	_, err := tx.Exec(ctx, `
		insert into public.scan_field_accuracy_events (
			scan_event_id,
			review_decision_id,
			user_id,
			field_name,
			predicted_value,
			corrected_value,
			error_source,
			confidence,
			calibrated_bucket,
			was_correct
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		emptyToNil(scanEventID),
		reviewDecisionID,
		userID,
		strings.TrimSpace(correction.FieldName),
		emptyToNil(correction.PredictedValue),
		emptyToNil(correction.CorrectedValue),
		normalizeErrorSource(correction.ErrorSource),
		correction.Confidence,
		confidenceBucket(correction.Confidence),
		strings.TrimSpace(correction.PredictedValue) == strings.TrimSpace(correction.CorrectedValue),
	)
	if err != nil {
		return fmt.Errorf("insert field accuracy event: %w", err)
	}
	return nil
}

func scanOCRText(input labelparser.ScanLabelInput, result labelparser.ScanLabelResult) string {
	if input.ClientOCR != nil && strings.TrimSpace(input.ClientOCR.Text) != "" {
		return strings.TrimSpace(input.ClientOCR.Text)
	}
	return strings.TrimSpace(result.RawText)
}

func scanFieldConfidences(result labelparser.ScanLabelResult) map[string]float64 {
	values := map[string]float64{
		"wash":                  result.Wash.Confidence,
		"bleach":                result.Bleach.Confidence,
		"drying":                result.Drying.Confidence,
		"ironing":               result.Ironing.Confidence,
		"professional_cleaning": result.ProfessionalCleaning.Confidence,
	}
	for key, value := range values {
		if value <= 0 {
			delete(values, key)
		}
	}
	return values
}

func sourceSignals(result labelparser.ScanLabelResult, reasons []string) []string {
	signals := append([]string{}, reasons...)
	if result.Provider != "" {
		signals = appendUnique(signals, "provider:"+result.Provider)
	}
	if result.Route != "" {
		signals = appendUnique(signals, "route:"+result.Route)
	}
	if result.CacheHit {
		signals = appendUnique(signals, "cache_hit")
	}
	return signals
}

func mustJSON(value any) string {
	if value == nil {
		return "{}"
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	if string(bytes) == "null" {
		return "{}"
	}
	return string(bytes)
}

func normalizedStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = appendUnique(normalized, value)
	}
	if normalized == nil {
		return []string{}
	}
	return normalized
}

func nullableInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func nullableFloat(value float64) *float64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func emptyToNil(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isSupportedAccuracyField(value string) bool {
	switch strings.TrimSpace(value) {
	case "wash_temperature", "wash_cycle", "hand_wash", "bleach", "tumble_dry", "natural_drying", "iron", "dry_clean", "wet_clean":
		return true
	default:
		return false
	}
}

func normalizeErrorSource(value string) string {
	switch strings.TrimSpace(value) {
	case "ocr", "symbol_detection", "rule_interpretation", "user_override":
		return value
	default:
		return "unknown"
	}
}

func confidenceBucket(value *float64) any {
	if value == nil {
		return nil
	}
	switch {
	case *value < 0.50:
		return "0.00-0.49"
	case *value < 0.70:
		return "0.50-0.69"
	case *value < 0.85:
		return "0.70-0.84"
	default:
		return "0.85-1.00"
	}
}
