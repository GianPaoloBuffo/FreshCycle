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

type ClientOCR struct {
	Text       string   `json:"text"`
	Confidence *float64 `json:"confidence,omitempty"`
}

type SymbolBoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type ClientSymbol struct {
	Name       string             `json:"name"`
	Class      string             `json:"class,omitempty"`
	Label      string             `json:"label,omitempty"`
	Confidence *float64           `json:"confidence,omitempty"`
	Box        *SymbolBoundingBox `json:"box,omitempty"`
}

type ScanLabelInput struct {
	ParseLabelInput
	ClientOCR       *ClientOCR
	ClientSymbols   []ClientSymbol
	DetectedSymbols []SymbolDetection
	ImageHash       string
}

type SymbolDetection struct {
	Class      string             `json:"class"`
	Label      string             `json:"label"`
	Confidence float64            `json:"confidence"`
	Box        *SymbolBoundingBox `json:"box,omitempty"`
	Source     string             `json:"source"`
}

type WashInstruction struct {
	Status            string  `json:"status"`
	MaxTemperatureC   *int    `json:"max_temperature_c"`
	Cycle             *string `json:"cycle"`
	Summary           string  `json:"summary"`
	Confidence        float64 `json:"confidence"`
	Explanation       string  `json:"explanation"`
	NeedsConfirmation bool    `json:"needs_confirmation"`
}

type BleachInstruction struct {
	Status            string  `json:"status"`
	Kind              *string `json:"kind"`
	Summary           string  `json:"summary"`
	Confidence        float64 `json:"confidence"`
	Explanation       string  `json:"explanation"`
	NeedsConfirmation bool    `json:"needs_confirmation"`
}

type DryingInstruction struct {
	Status            string  `json:"status"`
	Temperature       *string `json:"temperature"`
	Summary           string  `json:"summary"`
	Confidence        float64 `json:"confidence"`
	Explanation       string  `json:"explanation"`
	NeedsConfirmation bool    `json:"needs_confirmation"`
}

type IroningInstruction struct {
	Status            string  `json:"status"`
	Temperature       *string `json:"temperature"`
	Summary           string  `json:"summary"`
	Confidence        float64 `json:"confidence"`
	Explanation       string  `json:"explanation"`
	NeedsConfirmation bool    `json:"needs_confirmation"`
}

type ProfessionalCleaningInstruction struct {
	Status            string  `json:"status"`
	Method            *string `json:"method"`
	Summary           string  `json:"summary"`
	Confidence        float64 `json:"confidence"`
	Explanation       string  `json:"explanation"`
	NeedsConfirmation bool    `json:"needs_confirmation"`
}

type ScanLabelResult struct {
	Wash                  WashInstruction                 `json:"wash"`
	Bleach                BleachInstruction               `json:"bleach"`
	Drying                DryingInstruction               `json:"drying"`
	Ironing               IroningInstruction              `json:"ironing"`
	ProfessionalCleaning  ProfessionalCleaningInstruction `json:"professional_cleaning"`
	RawText               string                          `json:"raw_text"`
	Confidence            float64                         `json:"confidence"`
	Explanation           string                          `json:"explanation"`
	UncertainFields       []string                        `json:"uncertain_fields"`
	NeedsUserConfirmation bool                            `json:"needs_user_confirmation"`
	SymbolDetections      []SymbolDetection               `json:"symbol_detections"`
	Provider              string                          `json:"provider,omitempty"`
	Route                 string                          `json:"route,omitempty"`
	CacheHit              bool                            `json:"cache_hit"`
	ImageHash             string                          `json:"image_hash,omitempty"`
	PaidFallbackUsed      bool                            `json:"paid_fallback_used"`
	FallbackCallsAvoided  int                             `json:"fallback_calls_avoided,omitempty"`
	RoutingReasons        []string                        `json:"routing_reasons"`
}

type Scanner interface {
	ScanLabel(ctx context.Context, input ScanLabelInput) (ScanLabelResult, error)
}
