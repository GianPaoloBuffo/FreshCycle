package labelparser

import (
	"context"
	"strconv"
	"strings"
)

type SymbolDetector interface {
	DetectSymbols(ctx context.Context, input ScanLabelInput) ([]SymbolDetection, error)
}

type RuleSymbolDetector struct{}

func NewRuleSymbolDetector() RuleSymbolDetector {
	return RuleSymbolDetector{}
}

func (RuleSymbolDetector) DetectSymbols(_ context.Context, input ScanLabelInput) ([]SymbolDetection, error) {
	detections := make([]SymbolDetection, 0, len(input.ClientSymbols)+6)
	for _, symbol := range input.ClientSymbols {
		detection, ok := detectionFromClientSymbol(symbol)
		if !ok {
			continue
		}
		detections = appendUniqueSymbolDetection(detections, detection)
	}

	if input.ClientOCR != nil && strings.TrimSpace(input.ClientOCR.Text) != "" {
		confidence := 0.58
		if input.ClientOCR.Confidence != nil {
			confidence = normalizeConfidence(*input.ClientOCR.Confidence, 0.58) * 0.86
		}
		for _, class := range symbolClassesFromText(input.ClientOCR.Text) {
			detections = appendUniqueSymbolDetection(detections, SymbolDetection{
				Class:      class,
				Label:      laundrySymbolLabel(class, class),
				Confidence: normalizeConfidence(confidence, 0.58),
				Source:     "local_text_rules",
			})
		}
	}

	if detections == nil {
		detections = []SymbolDetection{}
	}
	return detections, nil
}

func detectionFromClientSymbol(symbol ClientSymbol) (SymbolDetection, bool) {
	class := strings.TrimSpace(symbol.Class)
	if normalizedClass, ok := NormalizeLaundrySymbolClass(firstNonEmpty(class, symbol.Name, symbol.Label)); ok {
		class = normalizedClass
	}
	if class == "" {
		return SymbolDetection{}, false
	}

	confidence := 0.65
	if symbol.Confidence != nil {
		confidence = normalizeConfidence(*symbol.Confidence, 0.65)
	}

	return SymbolDetection{
		Class:      class,
		Label:      firstNonEmpty(symbol.Label, laundrySymbolLabel(class, symbol.Name), symbol.Name),
		Confidence: confidence,
		Box:        symbol.Box,
		Source:     "client_detector",
	}, true
}

func symbolClassesFromText(text string) []string {
	normalized := normalizeForRules(text)
	compact := compactForRules(normalized)
	classes := make([]string, 0, 6)

	add := func(class string) {
		classes = appendUniqueString(classes, class)
	}

	switch {
	case containsAny(normalized, "do not wash", "dont wash", "no lavar") || containsAny(compact, "donotwash", "dontwash", "nolavar"):
		add(SymbolWashDoNotWash)
	case containsAny(normalized, "hand wash", "lavar a mano", "lavado a mano"):
		add(SymbolWashHand)
	case inferWashTemperature(normalized) != nil:
		add(mustWashTemperatureSymbol(normalized))
	case containsAny(normalized, "machine wash", "wash cold", "wash warm", "wash hot", "lavar a maquina"):
		add(SymbolWashTub)
	}

	switch {
	case containsAny(normalized, "do not bleach", "dont bleach", "no bleach", "no blanquear") || containsBleachProhibition(normalized, compact):
		add(SymbolBleachDoNotBleach)
	case containsAny(normalized, "non chlorine bleach", "non-chlorine bleach", "sin cloro"):
		add(SymbolBleachNonChlorine)
	case containsAny(normalized, "bleach allowed", "bleach when needed"):
		add(SymbolBleachAllowed)
	}

	switch {
	case containsAny(normalized, "do not tumble dry", "dont tumble dry", "no tumble dry", "no usar secadora") ||
		containsAny(compact, "donottumbledry", "donttumbledry", "notumbledry"):
		add(SymbolDryDoNotTumble)
	case containsAny(normalized, "tumble dry low", "low heat") || containsAny(compact, "tumbledrylow"):
		add(SymbolDryTumbleLow)
	case containsAny(normalized, "tumble dry medium") || containsAny(compact, "tumbledrymedium"):
		add(SymbolDryTumbleMedium)
	case containsAny(normalized, "tumble dry high") || containsAny(compact, "tumbledryhigh"):
		add(SymbolDryTumbleHigh)
	case containsAny(normalized, "tumble dry", "secadora"):
		add(SymbolDryTumble)
	case containsAny(normalized, "dry flat", "lay flat"):
		add(SymbolDryFlat)
	case containsAny(normalized, "line dry", "hang dry"):
		add(SymbolDryLine)
	}

	switch {
	case containsAny(normalized, "do not iron", "dont iron", "no iron", "no planchar") || containsAny(compact, "donotiron", "dontiron", "noiron", "noplanchar"):
		add(SymbolIronDoNotIron)
	case containsAny(normalized, "cool iron", "iron low", "low heat", "planchar a baja"):
		add(SymbolIronLow)
	case containsAny(normalized, "warm iron", "iron medium", "medium heat"):
		add(SymbolIronMedium)
	case containsAny(normalized, "hot iron", "iron high", "high heat"):
		add(SymbolIronHigh)
	case containsIronSignal(normalized, compact):
		add(SymbolIronAllowed)
	}

	switch {
	case containsAny(normalized, "do not dry clean", "dont dry clean", "no dry clean") || containsAny(compact, "donotdryclean", "dontdryclean", "nodryclean"):
		add(SymbolDryCleanDoNot)
	case containsAny(normalized, "dry clean only", "dry clean", "professional clean"):
		add(SymbolDryClean)
	}

	return classes
}

func mustWashTemperatureSymbol(normalized string) string {
	if temperature := inferWashTemperature(normalized); temperature != nil {
		return "wash_temperature_" + strconv.Itoa(*temperature)
	}
	return SymbolWashTub
}

func appendUniqueSymbolDetection(detections []SymbolDetection, next SymbolDetection) []SymbolDetection {
	next.Class = strings.TrimSpace(next.Class)
	if next.Class == "" {
		return detections
	}
	if next.Label == "" {
		next.Label = laundrySymbolLabel(next.Class, next.Class)
	}
	next.Confidence = normalizeConfidence(next.Confidence, 0.65)
	if next.Source == "" {
		next.Source = "local_detector"
	}

	for index, existing := range detections {
		if existing.Class != next.Class {
			continue
		}
		if next.Confidence > existing.Confidence {
			detections[index] = next
		}
		return detections
	}
	return append(detections, next)
}

func normalizeSymbolDetections(detections []SymbolDetection) []SymbolDetection {
	normalized := make([]SymbolDetection, 0, len(detections))
	for _, detection := range detections {
		class, ok := NormalizeLaundrySymbolClass(firstNonEmpty(detection.Class, detection.Label))
		if ok {
			detection.Class = class
		}
		normalized = appendUniqueSymbolDetection(normalized, detection)
	}
	if normalized == nil {
		return []SymbolDetection{}
	}
	return normalized
}
