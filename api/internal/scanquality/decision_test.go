package scanquality

import (
	"testing"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
)

func TestReviewReasonsIncludesLowConfidenceUncertaintyAndUncommonClasses(t *testing.T) {
	t.Parallel()

	score := 0.61
	reasons := ReviewReasons(labelparser.ScanLabelResult{
		Confidence:            0.58,
		NeedsUserConfirmation: true,
		UncertainFields:       []string{"wash", "drying"},
		PaidFallbackUsed:      true,
		SymbolDetections: []labelparser.SymbolDetection{
			{Class: "modifier_letter_p", Confidence: 0.9},
		},
	}, ScanRequestMetadata{
		CaptureQualityScore: &score,
		CaptureQualityHints: []string{"low_ocr_signal"},
	})

	for _, expected := range []string{
		"low_confidence",
		"needs_user_confirmation",
		"uncertain:wash",
		"uncertain:drying",
		"paid_fallback_used",
		"capture_quality",
		"capture_quality:low_ocr_signal",
		"uncommon_symbol_class",
	} {
		if !contains(reasons, expected) {
			t.Fatalf("expected reason %q in %#v", expected, reasons)
		}
	}
}

func TestActiveLearningPriorityElevatesAmbiguousFallbackScans(t *testing.T) {
	t.Parallel()

	priority := ActiveLearningPriority(labelparser.ScanLabelResult{
		Confidence:       0.42,
		PaidFallbackUsed: true,
	}, []string{"low_confidence", "needs_user_confirmation", "field_conflict", "uncommon_symbol_class"})

	if priority < 0.75 || priority > 1 {
		t.Fatalf("expected high bounded priority, got %f", priority)
	}
}

func TestNormalizeDecisionRejectsUnknownValues(t *testing.T) {
	t.Parallel()

	if NormalizeDecision("correct") != DecisionCorrect {
		t.Fatal("expected correct decision to normalize")
	}
	if NormalizeDecision("ship_it") != "" {
		t.Fatal("expected unknown decision to be rejected")
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
