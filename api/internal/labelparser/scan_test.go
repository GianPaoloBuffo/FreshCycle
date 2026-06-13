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
	if result.Wash.Confidence <= 0 || result.Wash.Explanation == "" {
		t.Fatalf("expected field confidence metadata, got %#v", result.Wash)
	}
}

func TestScanLabelMergesDetectedSymbolsIntoRules(t *testing.T) {
	t.Parallel()

	result, err := NewStubParser().ScanLabel(context.Background(), ScanLabelInput{
		ParseLabelInput: ParseLabelInput{
			Filename: "label.png",
			MIMEType: "image/png",
			Content:  []byte("image"),
		},
		DetectedSymbols: []SymbolDetection{
			{Class: SymbolWashHand, Confidence: 0.94, Source: "test_detector"},
			{Class: SymbolBleachDoNotBleach, Confidence: 0.91, Source: "test_detector"},
			{Class: SymbolDryDoNotTumble, Confidence: 0.89, Source: "test_detector"},
			{Class: SymbolIronLow, Confidence: 0.88, Source: "test_detector"},
		},
	})
	if err != nil {
		t.Fatalf("scan label: %v", err)
	}

	if result.Wash.Status != washStatusHand {
		t.Fatalf("expected hand wash from symbols, got %#v", result.Wash)
	}
	if result.Bleach.Status != bleachStatusDoNotBleach {
		t.Fatalf("expected do not bleach from symbols, got %#v", result.Bleach)
	}
	if result.Drying.Status != dryingStatusDoNotTumbleDry {
		t.Fatalf("expected do not tumble dry from symbols, got %#v", result.Drying)
	}
	if result.Ironing.Status != ironingStatusAllowed || result.Ironing.Temperature == nil || *result.Ironing.Temperature != "low" {
		t.Fatalf("expected low iron from symbols, got %#v", result.Ironing)
	}
}

func TestScanPipelineCachesByImageHash(t *testing.T) {
	t.Parallel()

	parser := &countingScanner{
		result: ScanLabelResult{
			Wash: WashInstruction{
				Status: washStatusMachine,
			},
			Bleach: BleachInstruction{
				Status: bleachStatusDoNotBleach,
			},
			Drying: DryingInstruction{
				Status: dryingStatusDoNotTumbleDry,
			},
			Ironing: IroningInstruction{
				Status: ironingStatusDoNotIron,
			},
			ProfessionalCleaning: ProfessionalCleaningInstruction{
				Status: cleaningStatusDoNotDryClean,
			},
			RawText:          "Machine wash. Do not bleach.",
			Confidence:       0.86,
			Provider:         "multimodal_fallback",
			Route:            "multimodal_fallback",
			PaidFallbackUsed: true,
		},
	}
	pipeline := NewScanPipeline(parser, NewRuleSymbolDetector(), NewMemoryScanCache(0, 8))
	input := ScanLabelInput{
		ParseLabelInput: ParseLabelInput{
			Filename: "same-label.jpg",
			MIMEType: "image/jpeg",
			Content:  []byte("same-image-bytes"),
		},
	}

	first, err := pipeline.ScanLabel(context.Background(), input)
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	second, err := pipeline.ScanLabel(context.Background(), input)
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}

	if parser.calls != 1 {
		t.Fatalf("expected parser to be called once, got %d", parser.calls)
	}
	if first.CacheHit {
		t.Fatal("did not expect first result to be a cache hit")
	}
	if !second.CacheHit {
		t.Fatal("expected second result to be a cache hit")
	}
	if second.FallbackCallsAvoided != 1 {
		t.Fatalf("expected avoided fallback call on cache hit, got %d", second.FallbackCallsAvoided)
	}
	if first.ImageHash == "" || second.ImageHash != first.ImageHash {
		t.Fatalf("expected stable image hashes, got first=%q second=%q", first.ImageHash, second.ImageHash)
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

type countingScanner struct {
	result ScanLabelResult
	calls  int
}

func (s *countingScanner) ParseLabel(_ context.Context, _ ParseLabelInput) (ParseLabelResult, error) {
	s.calls++
	return ParseLabelResult{
		NameSuggestion:  "Care Label",
		MachineWashable: true,
		RawLabelText:    "Machine wash",
	}, nil
}

func (s *countingScanner) ScanLabel(_ context.Context, _ ScanLabelInput) (ScanLabelResult, error) {
	s.calls++
	return s.result, nil
}
