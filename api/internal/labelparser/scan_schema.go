package labelparser

import (
	"encoding/json"
	"fmt"
	"strings"
)

func buildScanPrompt(input ScanLabelInput) string {
	context := make([]string, 0, 3)
	if input.ClientOCR != nil && strings.TrimSpace(input.ClientOCR.Text) != "" {
		payload, _ := json.Marshal(input.ClientOCR)
		context = append(context, "Client OCR hint: "+string(payload))
	}
	if len(input.ClientSymbols) > 0 {
		payload, _ := json.Marshal(input.ClientSymbols)
		context = append(context, "Client symbol hint: "+string(payload))
	}

	clientContext := "No client OCR or symbol hints were supplied."
	if len(context) > 0 {
		clientContext = strings.Join(context, "\n")
	}

	return fmt.Sprintf(
		"Extract Phase 1 garment care-label scanner details from this cropped label image. Return only JSON matching the provided schema. Use the enum status fields so a client can render the result without parsing prose. Be conservative: use unknown and add the field path to uncertain_fields when an instruction is not visible or conflicts. confidence must be 0.0 to 1.0. needs_user_confirmation should be true when confidence is below 0.75 or any field is uncertain. Include readable label text in raw_text. Filename: %s. MIME type: %s.\n%s",
		input.Filename,
		input.MIMEType,
		clientContext,
	)
}

var scanLabelResponseSchema = map[string]any{
	"type":                 "object",
	"additionalProperties": false,
	"properties": map[string]any{
		"wash": instructionObjectSchema(map[string]any{
			"status": map[string]any{
				"type": "string",
				"enum": []any{washStatusMachine, washStatusHand, washStatusDoNotWash, washStatusDryCleanOnly, washStatusUnknown},
			},
			"max_temperature_c": map[string]any{
				"type": []string{"integer", "null"},
			},
			"cycle": map[string]any{
				"type": []string{"string", "null"},
				"enum": []any{"normal", "delicate", "permanent_press", nil},
			},
			"summary": map[string]any{"type": "string"},
		}, []string{"status", "max_temperature_c", "cycle", "summary"}),
		"bleach": instructionObjectSchema(map[string]any{
			"status": map[string]any{
				"type": "string",
				"enum": []any{bleachStatusAllowed, bleachStatusNonChlorineOnly, bleachStatusDoNotBleach, bleachStatusUnknown},
			},
			"kind": map[string]any{
				"type": []string{"string", "null"},
				"enum": []any{"any", "non_chlorine", nil},
			},
			"summary": map[string]any{"type": "string"},
		}, []string{"status", "kind", "summary"}),
		"drying": instructionObjectSchema(map[string]any{
			"status": map[string]any{
				"type": "string",
				"enum": []any{dryingStatusTumbleDry, dryingStatusDoNotTumbleDry, dryingStatusLineDry, dryingStatusDryFlat, dryingStatusUnknown},
			},
			"temperature": map[string]any{
				"type": []string{"string", "null"},
				"enum": []any{"low", "medium", "high", nil},
			},
			"summary": map[string]any{"type": "string"},
		}, []string{"status", "temperature", "summary"}),
		"ironing": instructionObjectSchema(map[string]any{
			"status": map[string]any{
				"type": "string",
				"enum": []any{ironingStatusAllowed, ironingStatusDoNotIron, ironingStatusUnknown},
			},
			"temperature": map[string]any{
				"type": []string{"string", "null"},
				"enum": []any{"low", "medium", "high", nil},
			},
			"summary": map[string]any{"type": "string"},
		}, []string{"status", "temperature", "summary"}),
		"professional_cleaning": instructionObjectSchema(map[string]any{
			"status": map[string]any{
				"type": "string",
				"enum": []any{cleaningStatusDryCleanOnly, cleaningStatusDryClean, cleaningStatusDoNotDryClean, cleaningStatusUnknown},
			},
			"method": map[string]any{
				"type": []string{"string", "null"},
				"enum": []any{"dry_clean", "wet_clean", nil},
			},
			"summary": map[string]any{"type": "string"},
		}, []string{"status", "method", "summary"}),
		"raw_text": map[string]any{
			"type": "string",
		},
		"confidence": map[string]any{
			"type":    "number",
			"minimum": 0,
			"maximum": 1,
		},
		"explanation": map[string]any{
			"type": "string",
		},
		"uncertain_fields": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "string",
			},
		},
		"needs_user_confirmation": map[string]any{
			"type": "boolean",
		},
	},
	"required": []string{
		"wash",
		"bleach",
		"drying",
		"ironing",
		"professional_cleaning",
		"raw_text",
		"confidence",
		"explanation",
		"uncertain_fields",
		"needs_user_confirmation",
	},
}

func instructionObjectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
	}
}
