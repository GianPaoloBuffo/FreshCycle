package labelparser

import "context"

type ParseLabelInput struct {
	Filename string
	MIMEType string
	Content  []byte
}

type ParseLabelResult struct {
	NameSuggestion  string   `json:"name_suggestion"`
	FabricNotes     []string `json:"fabric_notes"`
	WashTempMax     *int     `json:"wash_temp_max"`
	MachineWashable bool     `json:"machine_washable"`
	TumbleDry       bool     `json:"tumble_dry"`
	DryCleanOnly    bool     `json:"dry_clean_only"`
	IronAllowed     bool     `json:"iron_allowed"`
	IronTemp        *string  `json:"iron_temp"`
	BleachAllowed   bool     `json:"bleach_allowed"`
	RawLabelText    string   `json:"raw_label_text"`
}

type Parser interface {
	ParseLabel(ctx context.Context, input ParseLabelInput) (ParseLabelResult, error)
}
