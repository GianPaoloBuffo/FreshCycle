package labelparser

import (
	"context"
	"testing"
)

func TestScanLabelUsesClientOCRWhenCareSignalsArePresent(t *testing.T) {
	t.Parallel()

	result, err := NewStubParser().ScanLabel(context.Background(), ScanLabelInput{
		ParseLabelInput: ParseLabelInput{
			Filename: "label.png",
			MIMEType: "image/png",
			Content:  []byte("image"),
		},
		ClientOCR: &ClientOCR{
			Text:       "Hand wash cold. Do not bleach. Dry flat. Do not iron.",
			Confidence: floatPtr(0.91),
		},
	})
	if err != nil {
		t.Fatalf("scan label: %v", err)
	}

	if result.Wash.Status != washStatusHand {
		t.Fatalf("expected hand wash from client OCR, got %#v", result.Wash)
	}
	if result.Drying.Status != dryingStatusDryFlat {
		t.Fatalf("expected dry flat from client OCR, got %#v", result.Drying)
	}
	if result.Confidence < 0.9 {
		t.Fatalf("expected client OCR confidence, got %f", result.Confidence)
	}
}

func TestDecodeScanLabelProviderOutputAcceptsLegacyPayloadConservatively(t *testing.T) {
	t.Parallel()

	result, err := decodeScanLabelProviderOutput(`{
		"name_suggestion":"Care Label",
		"fabric_notes":[],
		"wash_temp_max":30,
		"machine_washable":true,
		"tumble_dry":false,
		"dry_clean_only":false,
		"iron_allowed":true,
		"iron_temp":null,
		"bleach_allowed":false,
		"raw_label_text":"Machine wash cold. Do not bleach."
	}`)
	if err != nil {
		t.Fatalf("decode provider output: %v", err)
	}

	if result.Wash.Status != washStatusMachine {
		t.Fatalf("expected machine wash conversion, got %#v", result.Wash)
	}
	if result.Bleach.Status != bleachStatusDoNotBleach {
		t.Fatalf("expected conservative bleach conversion, got %#v", result.Bleach)
	}
	if result.Explanation == "" || result.UncertainFields == nil {
		t.Fatalf("expected default scan metadata, got %#v", result)
	}
	if !result.NeedsUserConfirmation {
		t.Fatal("expected legacy provider conversion to need user confirmation")
	}
}

func TestNormalizeScanLabelResultFillsMissingProviderMetadata(t *testing.T) {
	t.Parallel()

	result := normalizeScanLabelResult(ScanLabelResult{
		RawText: "MACHINE WASH 30C",
	})

	if result.Wash.Status != washStatusUnknown {
		t.Fatalf("expected missing wash object to default to unknown, got %#v", result.Wash)
	}
	if result.Confidence <= 0 {
		t.Fatalf("expected default confidence, got %f", result.Confidence)
	}
	if result.Explanation == "" {
		t.Fatal("expected default explanation")
	}
	if len(result.UncertainFields) == 0 {
		t.Fatal("expected missing fields to be marked uncertain")
	}
	if !result.NeedsUserConfirmation {
		t.Fatal("expected confirmation when fields are uncertain")
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
