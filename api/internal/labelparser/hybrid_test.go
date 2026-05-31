package labelparser

import (
	"context"
	"errors"
	"testing"
)

func TestHybridParserReturnsOCRWhenConfidenceIsGood(t *testing.T) {
	ocrResult := ParseLabelResult{NameSuggestion: "OCR", RawLabelText: "Machine wash cold"}
	fallback := &fakeParser{result: ParseLabelResult{NameSuggestion: "Gemini"}}
	parser := NewHybridParser(fakeOCRDetailsParser{
		details: OCRParseDetails{
			Result:            ocrResult,
			Text:              "Machine wash cold",
			AverageConfidence: 80,
			WordCount:         3,
			KeywordHits:       2,
			Evidence:          careRuleEvidence{HasCareSignal: true},
		},
	}, fallback)

	result, err := parser.ParseLabel(context.Background(), ParseLabelInput{})
	if err != nil {
		t.Fatalf("parse label: %v", err)
	}
	if result.NameSuggestion != "OCR" {
		t.Fatalf("expected OCR result, got %#v", result)
	}
	if fallback.called {
		t.Fatal("did not expect fallback to be called")
	}
}

func TestHybridParserUsesFallbackWhenOCRNeedsReview(t *testing.T) {
	fallback := &fakeParser{result: ParseLabelResult{NameSuggestion: "Gemini", RawLabelText: "Gemini text"}}
	parser := NewHybridParser(fakeOCRDetailsParser{
		details: OCRParseDetails{
			Result:            ParseLabelResult{NameSuggestion: "OCR", RawLabelText: "RN 123"},
			Text:              "RN 123",
			AverageConfidence: 90,
			WordCount:         2,
			FallbackReasons:   []string{"no_useful_text", "no_care_signal"},
		},
	}, fallback)

	result, err := parser.ParseLabel(context.Background(), ParseLabelInput{})
	if err != nil {
		t.Fatalf("parse label: %v", err)
	}
	if result.NameSuggestion != "Gemini" {
		t.Fatalf("expected fallback result, got %#v", result)
	}
	if !fallback.called {
		t.Fatal("expected fallback to be called")
	}
}

func TestHybridParserReturnsPartialOCRWhenFallbackFails(t *testing.T) {
	fallback := &fakeParser{err: errors.New("gemini down")}
	parser := NewHybridParser(fakeOCRDetailsParser{
		details: OCRParseDetails{
			Result:            ParseLabelResult{NameSuggestion: "OCR", RawLabelText: "Machine wash"},
			Text:              "Machine wash",
			AverageConfidence: 40,
			WordCount:         2,
			KeywordHits:       2,
			FallbackReasons:   []string{"low_ocr_confidence"},
		},
	}, fallback)

	result, err := parser.ParseLabel(context.Background(), ParseLabelInput{})
	if err != nil {
		t.Fatalf("parse label: %v", err)
	}
	if result.NameSuggestion != "OCR" {
		t.Fatalf("expected partial OCR result, got %#v", result)
	}
}

func TestHybridParserFailsWhenOCRAndFallbackFail(t *testing.T) {
	parser := NewHybridParser(
		fakeOCRDetailsParser{err: errors.New("ocr failed")},
		&fakeParser{err: errors.New("gemini failed")},
	)

	if _, err := parser.ParseLabel(context.Background(), ParseLabelInput{}); err == nil {
		t.Fatal("expected hybrid parser error")
	}
}

type fakeOCRDetailsParser struct {
	details OCRParseDetails
	err     error
}

func (p fakeOCRDetailsParser) ParseLabelWithDetails(_ context.Context, _ ParseLabelInput) (OCRParseDetails, error) {
	return p.details, p.err
}

type fakeParser struct {
	result ParseLabelResult
	err    error
	called bool
}

func (p *fakeParser) ParseLabel(_ context.Context, _ ParseLabelInput) (ParseLabelResult, error) {
	p.called = true
	return p.result, p.err
}
