package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	httpmiddleware "github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi/middleware"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/scanquality"
)

var (
	errInvalidClientOCR     = errors.New("invalid client OCR JSON")
	errInvalidClientSymbols = errors.New("invalid client symbols JSON")
)

func ScanLabel(parser labelparser.Parser, qualityStores ...scanquality.Store) http.HandlerFunc {
	var qualityStore scanquality.Store
	if len(qualityStores) > 0 {
		qualityStore = qualityStores[0]
	}

	return func(writer http.ResponseWriter, request *http.Request) {
		input, err := readScanLabelInput(request)
		if err != nil {
			recordScanFailure(request, qualityStore, input, readScanRequestMetadata(request), scanErrorCode(err))
			writeScanLabelError(writer, err)
			return
		}
		metadata := readScanRequestMetadata(request)

		result, err := labelparser.ScanLabel(request.Context(), parser, input)
		if err != nil {
			recordScanFailure(request, qualityStore, input, metadata, scanErrorCode(err))
			writeScanLabelError(writer, err)
			return
		}

		if qualityStore != nil {
			recordResult, err := recordScanSuccess(request, qualityStore, input, result, metadata)
			if err != nil {
				log.Printf("scan quality telemetry failed: %v", err)
			} else {
				result.ScanEventID = recordResult.ScanEventID
				result.ReviewQueueID = recordResult.ReviewQueueID
				result.ReviewReasons = recordResult.ReviewReasons
				result.ActiveLearningPriority = recordResult.ActiveLearningPriority
			}
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(result)
	}
}

func ScanReviewEvent(qualityStore scanquality.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if qualityStore == nil {
			writeJSONError(writer, http.StatusServiceUnavailable, "scan_quality_unavailable", "Scan review tracking is not configured yet.")
			return
		}

		user, ok := httpmiddleware.AuthenticatedUserFromContext(request.Context())
		if !ok || strings.TrimSpace(user.ID) == "" {
			writeJSONError(writer, http.StatusUnauthorized, "auth_required", "Sign in again before recording scan review feedback.")
			return
		}

		var payload scanReviewEventRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeJSONError(writer, http.StatusBadRequest, "invalid_review_event", "Review event must be valid JSON.")
			return
		}

		result, err := qualityStore.RecordReviewDecision(request.Context(), scanquality.ReviewDecisionInput{
			UserID:              user.ID,
			ScanEventID:         strings.TrimSpace(payload.ScanEventID),
			ReviewQueueID:       strings.TrimSpace(payload.ReviewQueueID),
			ImageHash:           strings.TrimSpace(payload.ImageHash),
			Decision:            strings.TrimSpace(payload.Decision),
			CorrectedFields:     payload.CorrectedFields,
			FieldCorrections:    payload.FieldCorrections,
			FinalUserCorrection: payload.FinalUserCorrection,
			Notes:               strings.TrimSpace(payload.Notes),
		})
		if err != nil {
			writeJSONError(writer, http.StatusBadRequest, "invalid_review_event", "FreshCycle could not record that scan review event.")
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(result)
	}
}

type scanReviewEventRequest struct {
	ScanEventID         string                        `json:"scan_event_id"`
	ReviewQueueID       string                        `json:"review_queue_id"`
	ImageHash           string                        `json:"image_hash"`
	Decision            string                        `json:"decision"`
	CorrectedFields     []string                      `json:"corrected_fields"`
	FieldCorrections    []scanquality.FieldCorrection `json:"field_corrections"`
	FinalUserCorrection map[string]any                `json:"final_user_correction"`
	Notes               string                        `json:"notes"`
}

func readScanLabelInput(request *http.Request) (labelparser.ScanLabelInput, error) {
	image, err := readLabelImage(request)
	if err != nil {
		return labelparser.ScanLabelInput{}, err
	}

	clientOCR, err := readClientOCR(request)
	if err != nil {
		return labelparser.ScanLabelInput{}, err
	}

	clientSymbols, err := readClientSymbols(request)
	if err != nil {
		return labelparser.ScanLabelInput{}, err
	}

	return labelparser.ScanLabelInput{
		ParseLabelInput: image,
		ClientOCR:       clientOCR,
		ClientSymbols:   clientSymbols,
	}, nil
}

func readScanRequestMetadata(request *http.Request) scanquality.ScanRequestMetadata {
	metadata := scanquality.ScanRequestMetadata{
		CaptureSource: optionalMultipartValue(request, "capture_source"),
		ImageScope:    optionalMultipartValue(request, "image_scope"),
	}
	if metadata.ImageScope == "" {
		metadata.ImageScope = "label_crop"
	}

	rawQuality := optionalMultipartValue(request, "capture_quality")
	if rawQuality == "" {
		return metadata
	}

	var payload struct {
		Score *float64 `json:"score"`
		Hints []struct {
			Issue string `json:"issue"`
		} `json:"hints"`
	}
	if err := json.Unmarshal([]byte(rawQuality), &payload); err != nil {
		return metadata
	}

	metadata.CaptureQualityScore = payload.Score
	for _, hint := range payload.Hints {
		if strings.TrimSpace(hint.Issue) != "" {
			metadata.CaptureQualityHints = append(metadata.CaptureQualityHints, strings.TrimSpace(hint.Issue))
		}
	}
	return metadata
}

func readClientOCR(request *http.Request) (*labelparser.ClientOCR, error) {
	raw := optionalMultipartValue(request, "client_ocr")
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var clientOCR labelparser.ClientOCR
	if err := json.Unmarshal([]byte(raw), &clientOCR); err != nil {
		return nil, errInvalidClientOCR
	}
	return &clientOCR, nil
}

func readClientSymbols(request *http.Request) ([]labelparser.ClientSymbol, error) {
	raw := optionalMultipartValue(request, "client_symbols")
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var symbols []labelparser.ClientSymbol
	if err := json.Unmarshal([]byte(raw), &symbols); err != nil {
		var wrapper struct {
			Symbols []labelparser.ClientSymbol `json:"symbols"`
		}
		if wrapperErr := json.Unmarshal([]byte(raw), &wrapper); wrapperErr != nil {
			return nil, errInvalidClientSymbols
		}
		symbols = wrapper.Symbols
	}

	normalized := make([]labelparser.ClientSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		if strings.TrimSpace(symbol.Name) == "" {
			continue
		}
		normalized = append(normalized, symbol)
	}
	return normalized, nil
}

func optionalMultipartValue(request *http.Request, field string) string {
	if request.MultipartForm == nil {
		return strings.TrimSpace(request.FormValue(field))
	}

	values := request.MultipartForm.Value[field]
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func writeScanLabelError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidClientOCR):
		writeJSONError(writer, http.StatusBadRequest, "invalid_client_ocr", "client_ocr must be valid JSON with readable label text.")
	case errors.Is(err, errInvalidClientSymbols):
		writeJSONError(writer, http.StatusBadRequest, "invalid_client_symbols", "client_symbols must be valid JSON.")
	default:
		writeParseLabelError(writer, err)
	}
}

func recordScanSuccess(request *http.Request, qualityStore scanquality.Store, input labelparser.ScanLabelInput, result labelparser.ScanLabelResult, metadata scanquality.ScanRequestMetadata) (scanquality.ScanRecordResult, error) {
	user, ok := httpmiddleware.AuthenticatedUserFromContext(request.Context())
	if !ok {
		return scanquality.ScanRecordResult{}, nil
	}

	return qualityStore.RecordScan(request.Context(), scanquality.ScanRecordInput{
		UserID:     user.ID,
		Outcome:    scanquality.OutcomeSuccess,
		Request:    metadata,
		ScanInput:  input,
		ScanResult: result,
	})
}

func recordScanFailure(request *http.Request, qualityStore scanquality.Store, input labelparser.ScanLabelInput, metadata scanquality.ScanRequestMetadata, errorCode string) {
	if qualityStore == nil {
		return
	}

	user, ok := httpmiddleware.AuthenticatedUserFromContext(request.Context())
	if !ok {
		return
	}

	if _, err := qualityStore.RecordScan(request.Context(), scanquality.ScanRecordInput{
		UserID:    user.ID,
		Outcome:   scanquality.OutcomeFailure,
		ErrorCode: errorCode,
		Request:   metadata,
		ScanInput: input,
	}); err != nil {
		log.Printf("scan quality failure telemetry failed: %v", err)
	}
}

func scanErrorCode(err error) string {
	switch {
	case errors.Is(err, errInvalidClientOCR):
		return "invalid_client_ocr"
	case errors.Is(err, errInvalidClientSymbols):
		return "invalid_client_symbols"
	case errors.Is(err, labelparser.ErrImageRequired):
		return "image_required"
	case errors.Is(err, labelparser.ErrUnsupportedImageType):
		return "unsupported_image_type"
	case errors.Is(err, labelparser.ErrProviderUnavailable):
		return "provider_unavailable"
	case errors.Is(err, labelparser.ErrUpstreamParseRejected):
		return "provider_error"
	default:
		return "parse_failed"
	}
}
