package labelparser

import (
	"context"
	"path/filepath"
	"strings"
)

type StubParser struct{}

func NewStubParser() StubParser {
	return StubParser{}
}

func (StubParser) ParseLabel(_ context.Context, input ParseLabelInput) (ParseLabelResult, error) {
	baseName := strings.TrimSpace(strings.TrimSuffix(filepath.Base(input.Filename), filepath.Ext(input.Filename)))
	baseName = strings.NewReplacer("_", " ", "-", " ").Replace(baseName)
	nameSuggestion := titleize(strings.ReplaceAll(baseName, "_", " "))
	if nameSuggestion == "" || nameSuggestion == "." {
		nameSuggestion = "Care Label Upload"
	}

	return ParseLabelResult{
		NameSuggestion:  nameSuggestion,
		FabricNotes:     []string{"Cotton blend", "Review label image for exact fabric percentages"},
		WashTempMax:     intPtr(30),
		MachineWashable: true,
		TumbleDry:       false,
		DryCleanOnly:    false,
		IronAllowed:     true,
		IronTemp:        stringPtr("low"),
		BleachAllowed:   false,
		RawLabelText:    "Machine wash cold. Do not bleach. Tumble dry low. Cool iron if needed.",
	}, nil
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func titleize(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}

	return strings.Join(parts, " ")
}
