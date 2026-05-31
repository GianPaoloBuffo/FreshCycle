package labelparser

import "testing"

func TestOpenAIResponsesResponseOutputTextUsesTopLevelHelper(t *testing.T) {
	response := openAIResponsesResponse{
		OutputText: `{"name_suggestion":"Linen shirt"}`,
	}

	if got := response.outputText(); got != `{"name_suggestion":"Linen shirt"}` {
		t.Fatalf("expected top-level output_text, got %q", got)
	}
}

func TestOpenAIResponsesResponseOutputTextFallsBackToMessageContent(t *testing.T) {
	response := openAIResponsesResponse{
		Output: []openAIResponseOutputItem{
			{
				Type: "message",
				Content: []openAIResponseContentItem{
					{
						Type: "output_text",
						Text: `{"name_suggestion":"Care label"}`,
					},
				},
			},
		},
	}

	if got := response.outputText(); got != `{"name_suggestion":"Care label"}` {
		t.Fatalf("expected nested output text, got %q", got)
	}
}

func TestOpenAIResponsesResponseOutputTextIgnoresNonTextContent(t *testing.T) {
	response := openAIResponsesResponse{
		Output: []openAIResponseOutputItem{
			{
				Type: "message",
				Content: []openAIResponseContentItem{
					{
						Type: "refusal",
						Text: "I cannot parse this image.",
					},
				},
			},
		},
	}

	if got := response.outputText(); got != "" {
		t.Fatalf("expected no output text, got %q", got)
	}
}
