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

func (p HybridParser) ScanLabel(ctx context.Context, input ScanLabelInput) (ScanLabelResult, error) {
	if p.primary == nil || p.fallback == nil {
		return ScanLabelResult{}, ErrProviderUnavailable
	}

	if result, ok := scanFromClientEvidence(input); ok && shouldAcceptLocalScan(result, 0.82, 3) {
		result.RoutingReasons = appendUniqueString(result.RoutingReasons, "high_confidence_client_evidence")
		log.Printf("hybrid scan-label parser used client evidence: confidence=%.2f known_fields=%d", result.Confidence, knownInstructionCount(result))
		return result, nil
	}

	ocrDetails, ocrErr := p.primary.ParseLabelWithDetails(ctx, input.ParseLabelInput)
	if ocrErr == nil && !ocrDetails.ShouldFallback() {
		log.Printf("hybrid scan-label parser used ocr: confidence=%.1f word_count=%d keyword_hits=%d", ocrDetails.AverageConfidence, ocrDetails.WordCount, ocrDetails.KeywordHits)
		result := scanFromOCRDetails(ocrDetails)
		result.RoutingReasons = appendUniqueString(result.RoutingReasons, "ocr_confidence_route")
		return result, nil
	}

	fallbackReason := "ocr_error"
	if ocrErr == nil {
		fallbackReason = strings.Join(ocrDetails.FallbackReasons, ",")
	}
	log.Printf("hybrid scan-label parser invoking fallback: reason=%s ocr_confidence=%.1f ocr_word_count=%d", fallbackReason, ocrDetails.AverageConfidence, ocrDetails.WordCount)

	fallbackResult, fallbackErr := scanWithFallbackParser(ctx, p.fallback, input)
	if fallbackErr == nil {
		fallbackResult.Explanation = strings.TrimSpace(fallbackResult.Explanation)
		if fallbackResult.Explanation == "" {
			fallbackResult.Explanation = "FreshCycle used multimodal fallback after OCR needed review."
		}
		fallbackResult.Provider = "multimodal_fallback"
		fallbackResult.Route = "multimodal_fallback"
		fallbackResult.PaidFallbackUsed = true
		fallbackResult.RoutingReasons = appendUniqueString(fallbackResult.RoutingReasons, fallbackReason)
		log.Printf("hybrid scan-label parser used fallback: reason=%s", fallbackReason)
		return normalizeScanLabelResult(fallbackResult), nil
	}

	if ocrErr == nil && ocrDetails.HasUsablePartial() {
		log.Printf("hybrid scan-label parser returning partial ocr after fallback failure: reason=%s fallback_error=%T", fallbackReason, fallbackErr)
		result := scanFromOCRDetails(ocrDetails)
		result.Route = "partial_server_ocr"
		result.RoutingReasons = appendUniqueString(result.RoutingReasons, "fallback_failed")
		return result, nil
	}

	if clientResult, ok := scanFromClientEvidence(input); ok {
		log.Printf("hybrid scan-label parser returning client evidence after fallback failure: reason=%s fallback_error=%T", fallbackReason, fallbackErr)
		clientResult.RoutingReasons = appendUniqueString(clientResult.RoutingReasons, "fallback_failed")
		return clientResult, nil
	}

	if ocrErr != nil {
		return ScanLabelResult{}, fmt.Errorf("%w: OCR failed and fallback failed", ErrUpstreamParseRejected)
	}

	return ScanLabelResult{}, fmt.Errorf("%w: fallback failed without usable OCR", ErrUpstreamParseRejected)
}

func scanWithFallbackParser(ctx context.Context, parser Parser, input ScanLabelInput) (ScanLabelResult, error) {
	if scanner, ok := parser.(Scanner); ok {
		return scanner.ScanLabel(ctx, input)
	}

	result, err := parser.ParseLabel(ctx, input.ParseLabelInput)
	if err != nil {
		return ScanLabelResult{}, err
	}
	return scanFromParseLabelResult(result, careRuleEvidence{}, "fallback parser", 0.62), nil
}
