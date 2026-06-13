package scanquality

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
)

const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"

	defaultReviewThreshold = 0.76
)

var uncommonSymbolClasses = map[string]struct{}{
	"wash_gentle":                   {},
	"wash_very_gentle":              {},
	"bleach_non_chlorine":           {},
	"dry_drip":                      {},
	"dry_shade":                     {},
	"professional_dry_clean_p":      {},
	"professional_dry_clean_f":      {},
	"professional_wet_clean":        {},
	"professional_do_not_wet_clean": {},
	"modifier_one_bar":              {},
	"modifier_two_bars":             {},
	"modifier_letter_p":             {},
	"modifier_letter_f":             {},
	"modifier_letter_w":             {},
}

func NormalizeEnvironment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case EnvironmentDebug:
		return EnvironmentDebug
	case EnvironmentTest:
		return EnvironmentTest
	default:
		return EnvironmentProduction
	}
}

func ReviewReasons(result labelparser.ScanLabelResult, request ScanRequestMetadata) []string {
	reasons := make([]string, 0, 8)

	if result.Confidence > 0 && result.Confidence < defaultReviewThreshold {
		reasons = appendUnique(reasons, "low_confidence")
	}
	if result.NeedsUserConfirmation {
		reasons = appendUnique(reasons, "needs_user_confirmation")
	}
	for _, field := range result.UncertainFields {
		field = strings.TrimSpace(field)
		if field != "" {
			reasons = appendUnique(reasons, "uncertain:"+field)
		}
	}
	if result.PaidFallbackUsed {
		reasons = appendUnique(reasons, "paid_fallback_used")
	}
	for _, reason := range result.RoutingReasons {
		normalized := strings.ToLower(strings.TrimSpace(reason))
		if strings.Contains(normalized, "conflict") {
			reasons = appendUnique(reasons, "field_conflict")
		}
		if strings.Contains(normalized, "fallback") {
			reasons = appendUnique(reasons, "fallback_route")
		}
	}
	if request.CaptureQualityScore != nil && *request.CaptureQualityScore < 0.72 {
		reasons = appendUnique(reasons, "capture_quality")
	}
	for _, hint := range request.CaptureQualityHints {
		hint = strings.TrimSpace(hint)
		if hint != "" {
			reasons = appendUnique(reasons, "capture_quality:"+hint)
		}
	}
	if len(UncommonClasses(result.SymbolDetections)) > 0 {
		reasons = appendUnique(reasons, "uncommon_symbol_class")
	}

	return reasons
}

func ActiveLearningPriority(result labelparser.ScanLabelResult, reasons []string) float64 {
	score := 0.10
	if result.Confidence > 0 && result.Confidence < defaultReviewThreshold {
		score += (defaultReviewThreshold - result.Confidence) * 0.75
	}
	for _, reason := range reasons {
		switch {
		case reason == "needs_user_confirmation":
			score += 0.18
		case reason == "paid_fallback_used" || reason == "fallback_route":
			score += 0.10
		case reason == "field_conflict":
			score += 0.20
		case reason == "uncommon_symbol_class":
			score += 0.16
		case strings.HasPrefix(reason, "uncertain:"):
			score += 0.05
		case strings.HasPrefix(reason, "capture_quality"):
			score += 0.04
		}
	}
	if score > 1 {
		return 1
	}
	if score < 0 {
		return 0
	}
	return score
}

func ShouldQueueForReview(result labelparser.ScanLabelResult, request ScanRequestMetadata) bool {
	return len(ReviewReasons(result, request)) > 0
}

func UncommonClasses(detections []labelparser.SymbolDetection) []string {
	classes := make([]string, 0, len(detections))
	for _, detection := range detections {
		class := strings.TrimSpace(detection.Class)
		if _, ok := uncommonSymbolClasses[class]; ok {
			classes = appendUnique(classes, class)
		}
	}
	return classes
}

func SymbolClasses(detections []labelparser.SymbolDetection) []string {
	classes := make([]string, 0, len(detections))
	for _, detection := range detections {
		class := strings.TrimSpace(detection.Class)
		if class != "" {
			classes = appendUnique(classes, class)
		}
	}
	return classes
}

func ImageHash(input labelparser.ScanLabelInput) string {
	if input.ImageHash != "" {
		return input.ImageHash
	}
	if len(input.Content) == 0 {
		return ""
	}

	hash := sha256.New()
	_, _ = hash.Write([]byte(input.MIMEType))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(input.Content)
	return hex.EncodeToString(hash.Sum(nil))
}

func TextHash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func NormalizeDecision(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case DecisionAccept:
		return DecisionAccept
	case DecisionCorrect:
		return DecisionCorrect
	case DecisionNeedsLabel:
		return DecisionNeedsLabel
	case DecisionDiscard:
		return DecisionDiscard
	case DecisionPrivacyDelete:
		return DecisionPrivacyDelete
	default:
		return ""
	}
}

func DecisionStatus(decision string) string {
	switch NormalizeDecision(decision) {
	case DecisionAccept:
		return "accepted"
	case DecisionCorrect:
		return "corrected"
	case DecisionNeedsLabel:
		return "needs_label"
	case DecisionDiscard:
		return "discarded"
	case DecisionPrivacyDelete:
		return "privacy_deleted"
	default:
		return "queued"
	}
}

func appendUnique(values []string, next string) []string {
	next = strings.TrimSpace(next)
	if next == "" {
		return values
	}
	for _, existing := range values {
		if existing == next {
			return values
		}
	}
	return append(values, next)
}
