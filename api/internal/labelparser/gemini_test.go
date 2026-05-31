package labelparser

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiParserSendsInlineImageAndDecodesStructuredOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/models/gemini-test:generateContent" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		if request.Header.Get("x-goog-api-key") != "test-key" {
			t.Fatalf("unexpected API key header %q", request.Header.Get("x-goog-api-key"))
		}

		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		generationConfig := payload["generationConfig"].(map[string]any)
		if generationConfig["responseMimeType"] != "application/json" {
			t.Fatalf("expected JSON response mime type, got %#v", generationConfig["responseMimeType"])
		}
		if generationConfig["responseJsonSchema"] == nil {
			t.Fatal("expected structured response schema")
		}

		contents := payload["contents"].([]any)
		parts := contents[0].(map[string]any)["parts"].([]any)
		inlineData := parts[1].(map[string]any)["inline_data"].(map[string]any)
		if inlineData["mime_type"] != "image/jpeg" || inlineData["data"] == "" {
			t.Fatalf("expected inline image data, got %#v", inlineData)
		}

		writeGeminiResponse(t, writer, `{
			"name_suggestion":"Linen Shirt",
			"fabric_notes":["55% linen"],
			"wash_temp_max":30,
			"machine_washable":true,
			"tumble_dry":false,
			"dry_clean_only":false,
			"iron_allowed":true,
			"iron_temp":"low",
			"bleach_allowed":false,
			"raw_label_text":"Machine wash cold"
		}`)
	}))
	defer server.Close()

	parser := NewGeminiParser("test-key", "gemini-test", server.URL)
	result, err := parser.ParseLabel(context.Background(), ParseLabelInput{
		Filename: "label.jpg",
		MIMEType: "image/jpeg",
		Content:  []byte("image-bytes"),
	})
	if err != nil {
		t.Fatalf("parse label: %v", err)
	}

	if result.NameSuggestion != "Linen Shirt" || result.WashTempMax == nil || *result.WashTempMax != 30 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestGeminiParserRejectsMalformedStructuredOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeGeminiResponse(t, writer, `{"name_suggestion":`)
	}))
	defer server.Close()

	parser := NewGeminiParser("test-key", "gemini-test", server.URL)
	if _, err := parser.ParseLabel(context.Background(), ParseLabelInput{MIMEType: "image/jpeg", Content: []byte("image")}); err == nil {
		t.Fatal("expected malformed structured output error")
	}
}

func TestGeminiParserMapsUpstreamErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	parser := NewGeminiParser("test-key", "gemini-test", server.URL)
	_, err := parser.ParseLabel(context.Background(), ParseLabelInput{MIMEType: "image/jpeg", Content: []byte("image")})
	if !errors.Is(err, ErrUpstreamParseRejected) {
		t.Fatalf("expected upstream parse rejection, got %v", err)
	}
}

func writeGeminiResponse(t *testing.T, writer http.ResponseWriter, text string) {
	t.Helper()

	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{"text": text},
					},
				},
			},
		},
	}); err != nil {
		t.Fatalf("write response: %v", err)
	}
}
