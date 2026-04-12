package labelparser

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1/responses"

type OpenAIParser struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOpenAIParser(apiKey string, model string, baseURL string) OpenAIParser {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultOpenAIBaseURL
	}

	return OpenAIParser{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: strings.TrimSpace(baseURL),
		model:   strings.TrimSpace(model),
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (p OpenAIParser) ParseLabel(ctx context.Context, input ParseLabelInput) (ParseLabelResult, error) {
	if strings.TrimSpace(p.apiKey) == "" || strings.TrimSpace(p.model) == "" {
		return ParseLabelResult{}, ErrProviderUnavailable
	}

	requestBody := openAIResponsesRequest{
		Model: p.model,
		Input: []openAIInputItem{
			{
				Role: "user",
				Content: []openAIContentItem{
					{
						Type: "input_text",
						Text: buildOpenAIPrompt(input),
					},
					{
						Type:     "input_image",
						ImageURL: fmt.Sprintf("data:%s;base64,%s", input.MIMEType, base64.StdEncoding.EncodeToString(input.Content)),
					},
				},
			},
		},
		Text: openAITextConfig{
			Format: openAITextFormat{
				Type:   "json_schema",
				Name:   "garment_label_parse",
				Strict: true,
				Schema: openAIResponseSchema,
			},
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return ParseLabelResult{}, fmt.Errorf("marshal OpenAI request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(payload))
	if err != nil {
		return ParseLabelResult{}, fmt.Errorf("build OpenAI request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+p.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := p.httpClient.Do(request)
	if err != nil {
		return ParseLabelResult{}, fmt.Errorf("call OpenAI responses API: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return ParseLabelResult{}, fmt.Errorf("read OpenAI response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return ParseLabelResult{}, fmt.Errorf("%w: OpenAI returned %s", ErrUpstreamParseRejected, response.Status)
	}

	var parsed openAIResponsesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ParseLabelResult{}, fmt.Errorf("decode OpenAI response: %w", err)
	}

	outputText := strings.TrimSpace(parsed.OutputText)
	if outputText == "" {
		return ParseLabelResult{}, fmt.Errorf("%w: missing output_text", ErrUpstreamParseRejected)
	}

	var result ParseLabelResult
	if err := json.Unmarshal([]byte(outputText), &result); err != nil {
		return ParseLabelResult{}, fmt.Errorf("decode OpenAI structured output: %w", err)
	}

	return result, nil
}

func buildOpenAIPrompt(input ParseLabelInput) string {
	return fmt.Sprintf(
		"Extract garment care-label details from this image. Return only the structured fields requested by the schema. Be conservative: if the wash temperature or iron temperature is not visible, set it to null. Use false for boolean care instructions unless the symbol or text clearly indicates the instruction applies. Include any readable text from the label in raw_label_text. Filename: %s. MIME type: %s.",
		input.Filename,
		input.MIMEType,
	)
}

type openAIResponsesRequest struct {
	Model string            `json:"model"`
	Input []openAIInputItem `json:"input"`
	Text  openAITextConfig  `json:"text"`
}

type openAIInputItem struct {
	Role    string              `json:"role"`
	Content []openAIContentItem `json:"content"`
}

type openAIContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type openAITextConfig struct {
	Format openAITextFormat `json:"format"`
}

type openAITextFormat struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type openAIResponsesResponse struct {
	OutputText string `json:"output_text"`
}

var openAIResponseSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"properties": map[string]any{
		"name_suggestion": map[string]any{
			"type":        "string",
			"description": "A short human-readable garment name inferred from the visible label context.",
		},
		"fabric_notes": map[string]any{
			"type":        "array",
			"description": "Short fabric composition notes or uncertainty cues from the care label.",
			"items": map[string]any{
				"type": "string",
			},
		},
		"wash_temp_max": map[string]any{
			"type":        []string{"integer", "null"},
			"description": "Maximum machine wash temperature in Celsius when visible.",
		},
		"machine_washable": map[string]any{
			"type": "boolean",
		},
		"tumble_dry": map[string]any{
			"type": "boolean",
		},
		"dry_clean_only": map[string]any{
			"type": "boolean",
		},
		"iron_allowed": map[string]any{
			"type": "boolean",
		},
		"iron_temp": map[string]any{
			"type":        []string{"string", "null"},
			"enum":        []any{"low", "medium", "high", nil},
			"description": "Iron temperature band when visible.",
		},
		"bleach_allowed": map[string]any{
			"type": "boolean",
		},
		"raw_label_text": map[string]any{
			"type":        "string",
			"description": "Readable text transcribed from the label image.",
		},
	},
	"required": []string{
		"name_suggestion",
		"fabric_notes",
		"wash_temp_max",
		"machine_washable",
		"tumble_dry",
		"dry_clean_only",
		"iron_allowed",
		"iron_temp",
		"bleach_allowed",
		"raw_label_text",
	},
}
