package labelparser

import (
	"context"
	"testing"
)

func TestNormalizeLaundrySymbolClassCoversCommonVariants(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"wash tub 30 underline":       "wash_temperature_30",
		"crossed triangle":            SymbolBleachDoNotBleach,
		"do not tumble dry":           SymbolDryDoNotTumble,
		"one dot iron":                SymbolIronLow,
		"circle P professional clean": SymbolDryCleanP,
	}

	for input, expected := range cases {
		got, ok := NormalizeLaundrySymbolClass(input)
		if !ok {
			t.Fatalf("expected %q to normalize", input)
		}
		if got != expected {
			t.Fatalf("expected %q to normalize to %q, got %q", input, expected, got)
		}
	}
}

func TestRuleSymbolDetectorReturnsClassConfidenceAndBoxes(t *testing.T) {
	t.Parallel()

	confidence := 0.93
	detector := NewRuleSymbolDetector()
	detections, err := detector.DetectSymbols(context.Background(), ScanLabelInput{
		ClientSymbols: []ClientSymbol{
			{
				Name:       "do_not_bleach",
				Confidence: &confidence,
				Box:        &SymbolBoundingBox{X: 1, Y: 2, Width: 3, Height: 4},
			},
		},
	})
	if err != nil {
		t.Fatalf("detect symbols: %v", err)
	}

	if len(detections) != 1 {
		t.Fatalf("expected one detection, got %#v", detections)
	}
	if detections[0].Class != SymbolBleachDoNotBleach {
		t.Fatalf("expected do-not-bleach class, got %#v", detections[0])
	}
	if detections[0].Confidence != confidence {
		t.Fatalf("expected confidence %.2f, got %.2f", confidence, detections[0].Confidence)
	}
	if detections[0].Box == nil || detections[0].Box.Width != 3 {
		t.Fatalf("expected bounding box to be preserved, got %#v", detections[0].Box)
	}
}
