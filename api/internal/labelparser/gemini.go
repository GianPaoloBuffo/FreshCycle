package labelparser

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type GeminiParser struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewGeminiParser(apiKey string, model string, baseURL string) GeminiParser {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultGeminiBaseURL
	}

	return GeminiParser{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		model:   strings.TrimSpace(model),
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (p GeminiParser) ParseLabel(ctx context.Context, input ParseLabelInput) (ParseLabelResult, error) {
	if strings.TrimSpace(p.apiKey) == "" || strings.TrimSpace(p.model) == "" {
		return ParseLabelResult{}, ErrProviderUnavailable
	}

	requestBody := geminiGenerateRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{
						Text: buildGeminiPrompt(input),
					},
					{
						InlineData: &geminiInlineData{
							MIMEType: input.MIMEType,
							Data:     base64.StdEncoding.EncodeToString(input.Content),
						},
					},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   geminiResponseSchema,
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return ParseLabelResult{}, fmt.Errorf("marshal Gemini request: %w", err)
	}

	requestURL := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, strings.TrimPrefix(p.model, "models/"))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return ParseLabelResult{}, fmt.Errorf("build Gemini request: %w", err)
	}
	request.Header.Set("x-goog-api-key", p.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := p.httpClient.Do(request)
	if err != nil {
		return ParseLabelResult{}, fmt.Errorf("call Gemini generateContent API: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return ParseLabelResult{}, fmt.Errorf("read Gemini response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		log.Printf("gemini label parser rejected request: status=%s body=%q", response.Status, truncateForLog(string(body), 1000))
		return ParseLabelResult{}, fmt.Errorf("%w: Gemini returned %s", ErrUpstreamParseRejected, response.Status)
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ParseLabelResult{}, fmt.Errorf("decode Gemini response: %w", err)
	}

	outputText := strings.TrimSpace(parsed.outputText())
	if outputText == "" {
		log.Printf("gemini label parser response missing output text: body=%q", truncateForLog(string(body), 1000))
		return ParseLabelResult{}, fmt.Errorf("%w: missing Gemini output text", ErrUpstreamParseRejected)
	}

	var result ParseLabelResult
	if err := json.Unmarshal([]byte(outputText), &result); err != nil {
		return ParseLabelResult{}, fmt.Errorf("decode Gemini structured output: %w", err)
	}

	return result, nil
}

func (p GeminiParser) ScanLabel(ctx context.Context, input ScanLabelInput) (ScanLabelResult, error) {
	if strings.TrimSpace(p.apiKey) == "" || strings.TrimSpace(p.model) == "" {
		return ScanLabelResult{}, ErrProviderUnavailable
	}

	requestBody := geminiGenerateRequest{
		Contents: []geminiContent{
			{
				Role: "user",
				Parts: []geminiPart{
					{
						Text: buildScanPrompt(input),
					},
					{
						InlineData: &geminiInlineData{
							MIMEType: input.MIMEType,
							Data:     base64.StdEncoding.EncodeToString(input.Content),
						},
					},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMIMEType: "application/json",
			ResponseSchema:   scanLabelResponseSchema,
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return ScanLabelResult{}, fmt.Errorf("marshal Gemini scan-label request: %w", err)
	}

	requestURL := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, strings.TrimPrefix(p.model, "models/"))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return ScanLabelResult{}, fmt.Errorf("build Gemini scan-label request: %w", err)
	}
	request.Header.Set("x-goog-api-key", p.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := p.httpClient.Do(request)
	if err != nil {
		return ScanLabelResult{}, fmt.Errorf("call Gemini generateContent API for scan-label: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return ScanLabelResult{}, fmt.Errorf("read Gemini scan-label response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		log.Printf("gemini scan-label parser rejected request: status=%s body=%q", response.Status, truncateForLog(string(body), 1000))
		return ScanLabelResult{}, fmt.Errorf("%w: Gemini returned %s", ErrUpstreamParseRejected, response.Status)
	}

	var parsed geminiGenerateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ScanLabelResult{}, fmt.Errorf("decode Gemini scan-label response: %w", err)
	}

	outputText := strings.TrimSpace(parsed.outputText())
	if outputText == "" {
		log.Printf("gemini scan-label parser response missing output text: body=%q", truncateForLog(string(body), 1000))
		return ScanLabelResult{}, fmt.Errorf("%w: missing Gemini output text", ErrUpstreamParseRejected)
	}

	return decodeScanLabelProviderOutput(outputText)
}

func buildGeminiPrompt(input ParseLabelInput) string {
	return fmt.Sprintf(
		"Extract garment care-label details from this image. Return only JSON matching the provided schema. Be conservative: if the wash temperature or iron temperature is not visible, set it to null. Use false for boolean care instructions unless the symbol or text clearly indicates the instruction applies. Include readable text from the label in raw_label_text. Filename: %s. MIME type: %s.",
		input.Filename,
		input.MIMEType,
	)
}

type geminiGenerateRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}

type geminiInlineData struct {
	MIMEType string `json:"mime_type"`
	Data     string `json:"data"`
}

type geminiGenerationConfig struct {
	ResponseMIMEType string         `json:"responseMimeType"`
	ResponseSchema   map[string]any `json:"responseJsonSchema"`
}

type geminiGenerateResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

func (r geminiGenerateResponse) outputText() string {
	for _, candidate := range r.Candidates {
		for _, part := range candidate.Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				return part.Text
			}
		}
	}
	return ""
}

var geminiResponseSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"name_suggestion": map[string]any{
			"type": "string",
		},
		"fabric_notes": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
		"wash_temp_max": map[string]any{
			"type": []string{"integer", "null"},
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
			"type": []string{"string", "null"},
			"enum": []any{"low", "medium", "high", nil},
		},
		"bleach_allowed": map[string]any{
			"type": "boolean",
		},
		"raw_label_text": map[string]any{
			"type": "string",
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
