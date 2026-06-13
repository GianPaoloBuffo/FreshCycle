package scanquality

import (
	"context"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
)

const (
	EnvironmentProduction = "production"
	EnvironmentDebug      = "debug"
	EnvironmentTest       = "test"

	DecisionAccept        = "accept"
	DecisionCorrect       = "correct"
	DecisionNeedsLabel    = "needs_label"
	DecisionDiscard       = "discard"
	DecisionPrivacyDelete = "privacy_delete"
)

type Store interface {
	RecordScan(ctx context.Context, input ScanRecordInput) (ScanRecordResult, error)
	RecordReviewDecision(ctx context.Context, input ReviewDecisionInput) (ReviewDecisionResult, error)
}

type ScanRecordInput struct {
	UserID      string
	Outcome     string
	ErrorCode   string
	Request     ScanRequestMetadata
	ScanInput   labelparser.ScanLabelInput
	ScanResult  labelparser.ScanLabelResult
	Environment string
}

type ScanRequestMetadata struct {
	CaptureSource       string
	ImageScope          string
	CaptureQualityScore *float64
	CaptureQualityHints []string
	CroppedImagePath    string
}

type ScanRecordResult struct {
	ScanEventID            string
	ReviewQueueID          string
	ReviewReasons          []string
	ActiveLearningPriority float64
}

type ReviewDecisionInput struct {
	UserID              string
	ScanEventID         string
	ReviewQueueID       string
	ImageHash           string
	Decision            string
	CorrectedFields     []string
	FieldCorrections    []FieldCorrection
	FinalUserCorrection map[string]any
	Notes               string
}

type FieldCorrection struct {
	FieldName      string   `json:"field_name"`
	PredictedValue string   `json:"predicted_value"`
	CorrectedValue string   `json:"corrected_value"`
	ErrorSource    string   `json:"error_source"`
	Confidence     *float64 `json:"confidence,omitempty"`
}

type ReviewDecisionResult struct {
	ReviewDecisionID    string `json:"review_decision_id"`
	ReviewQueueID       string `json:"review_queue_id,omitempty"`
	AnnotationExampleID string `json:"annotation_example_id,omitempty"`
}
