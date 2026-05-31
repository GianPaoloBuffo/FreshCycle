package labelparser

import (
	"fmt"
	"strings"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/config"
)

func NewParser(cfg config.Config) (Parser, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.LabelParserProvider)) {
	case "", "stub":
		return NewStubParser(), nil
	case "openai":
		return NewOpenAIParser(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.OpenAIBaseURL), nil
	case "ocr":
		return NewOCRParser(cfg.OCRTesseractPath, cfg.OCRLanguages), nil
	case "hybrid":
		fallbackProvider := strings.ToLower(strings.TrimSpace(cfg.LabelParserFallbackProvider))
		if fallbackProvider != "gemini" {
			return nil, fmt.Errorf("unsupported LABEL_PARSER_FALLBACK_PROVIDER %q", cfg.LabelParserFallbackProvider)
		}

		return NewHybridParser(
			NewOCRParser(cfg.OCRTesseractPath, cfg.OCRLanguages),
			NewGeminiParser(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiBaseURL),
		), nil
	default:
		return nil, fmt.Errorf("unsupported LABEL_PARSER_PROVIDER %q", cfg.LabelParserProvider)
	}
}
