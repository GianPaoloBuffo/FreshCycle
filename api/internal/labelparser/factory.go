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
	default:
		return nil, fmt.Errorf("unsupported LABEL_PARSER_PROVIDER %q", cfg.LabelParserProvider)
	}
}
