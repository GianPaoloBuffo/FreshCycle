package labelparser

import (
	"fmt"
	"strings"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/config"
)

func NewParser(cfg config.Config) (Parser, error) {
	var parser Parser
	switch strings.ToLower(strings.TrimSpace(cfg.LabelParserProvider)) {
	case "", "stub":
		parser = NewStubParser()
	case "openai":
		parser = NewOpenAIParser(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIBaseURL)
	case "ocr":
		parser = NewOCRParser(cfg.OCRTesseractPath, cfg.OCRLanguages)
	case "hybrid":
		fallbackProvider := strings.ToLower(strings.TrimSpace(cfg.LabelParserFallbackProvider))
		if fallbackProvider != "gemini" {
			return nil, fmt.Errorf("unsupported LABEL_PARSER_FALLBACK_PROVIDER %q", cfg.LabelParserFallbackProvider)
		}

		parser = NewHybridParser(
			NewOCRParser(cfg.OCRTesseractPath, cfg.OCRLanguages),
			NewGeminiParser(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiBaseURL),
		)
	default:
		return nil, fmt.Errorf("unsupported LABEL_PARSER_PROVIDER %q", cfg.LabelParserProvider)
	}

	var detector SymbolDetector
	if cfg.SymbolDetectorEnabled {
		detector = NewRuleSymbolDetector()
	}

	var cache ScanCache
	if cfg.ScanCacheEnabled {
		cache = NewMemoryScanCache(cfg.ScanCacheTTL, cfg.ScanCacheMaxEntries)
	}

	if detector == nil && cache == nil {
		return parser, nil
	}
	return NewScanPipeline(parser, detector, cache), nil
}
