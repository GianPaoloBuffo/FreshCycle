package labelparser

import (
	"context"
	"fmt"
	"log"
	"strings"
)

type OCRDetailsParser interface {
	ParseLabelWithDetails(ctx context.Context, input ParseLabelInput) (OCRParseDetails, error)
}

type HybridParser struct {
	primary  OCRDetailsParser
	fallback Parser
}

func NewHybridParser(primary OCRDetailsParser, fallback Parser) HybridParser {
	return HybridParser{
		primary:  primary,
		fallback: fallback,
	}
}

func (p HybridParser) ParseLabel(ctx context.Context, input ParseLabelInput) (ParseLabelResult, error) {
	if p.primary == nil || p.fallback == nil {
		return ParseLabelResult{}, ErrProviderUnavailable
	}

	ocrDetails, ocrErr := p.primary.ParseLabelWithDetails(ctx, input)
	if ocrErr == nil && !ocrDetails.ShouldFallback() {
		log.Printf("hybrid label parser used ocr: confidence=%.1f word_count=%d keyword_hits=%d", ocrDetails.AverageConfidence, ocrDetails.WordCount, ocrDetails.KeywordHits)
		return ocrDetails.Result, nil
	}

	fallbackReason := "ocr_error"
	if ocrErr == nil {
		fallbackReason = strings.Join(ocrDetails.FallbackReasons, ",")
	}
	log.Printf("hybrid label parser invoking fallback: reason=%s ocr_confidence=%.1f ocr_word_count=%d", fallbackReason, ocrDetails.AverageConfidence, ocrDetails.WordCount)

	fallbackResult, fallbackErr := p.fallback.ParseLabel(ctx, input)
	if fallbackErr == nil {
		log.Printf("hybrid label parser used fallback: reason=%s", fallbackReason)
		return fallbackResult, nil
	}

	if ocrErr == nil && ocrDetails.HasUsablePartial() {
		log.Printf("hybrid label parser returning partial ocr after fallback failure: reason=%s fallback_error=%T", fallbackReason, fallbackErr)
		return ocrDetails.Result, nil
	}

	if ocrErr != nil {
		return ParseLabelResult{}, fmt.Errorf("%w: OCR failed and fallback failed", ErrUpstreamParseRejected)
	}

	return ParseLabelResult{}, fmt.Errorf("%w: fallback failed without usable OCR", ErrUpstreamParseRejected)
}
