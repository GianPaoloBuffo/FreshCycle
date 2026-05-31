package labelparser

import (
	"regexp"
	"strconv"
	"strings"
)

type careRuleEvidence struct {
	HasCareSignal bool
	Conflicting   bool
}

func parseCareLabelText(text string) (ParseLabelResult, careRuleEvidence) {
	rawText := normalizeOCRText(text)
	normalized := normalizeForRules(rawText)

	washTemp := inferWashTemperature(normalized)
	negativeWash := containsAny(normalized, "do not wash", "dont wash", "no lavar")
	handWashOnly := containsAny(normalized, "hand wash only", "hand wash", "lavar a mano", "lavado a mano")
	dryCleanOnly := containsAny(normalized, "dry clean only", "professional dry clean only", "limpieza en seco solamente", "solo limpieza en seco")
	doNotDryClean := containsAny(normalized, "do not dry clean", "dont dry clean", "no lavar en seco", "no limpieza en seco")

	machineWashSignal := containsAny(normalized, "machine wash", "machine washable", "wash cold", "cold wash", "wash warm", "wash hot", "lavar a maquina", "lavado a maquina")
	machineWashable := !negativeWash && !dryCleanOnly && !handWashOnly && (machineWashSignal || washTemp != nil)

	negativeTumble := containsAny(normalized, "do not tumble dry", "dont tumble dry", "no tumble dry", "avoid tumble dry", "avoid tumble drying", "no usar secadora")
	positiveTumble := containsAny(normalized, "tumble dry", "secadora")
	tumbleDry := positiveTumble && !negativeTumble

	negativeBleach := containsAny(normalized, "do not bleach", "dont bleach", "no bleach", "no usar lejia", "no usar blanqueador", "no blanquear")
	positiveBleach := containsAny(normalized, "bleach allowed", "bleach when needed", "non chlorine bleach", "non-chlorine bleach", "only non chlorine bleach", "blanqueador sin cloro")
	bleachAllowed := positiveBleach && !negativeBleach

	negativeIron := containsAny(normalized, "do not iron", "dont iron", "no iron", "no planchar")
	positiveIron := containsAny(normalized, "iron", "cool iron", "warm iron", "hot iron", "planchar")
	ironAllowed := positiveIron && !negativeIron
	ironTemp := inferIronTemperature(normalized)
	if !ironAllowed {
		ironTemp = nil
	}

	fabricNotes := extractFabricNotes(normalized)
	hasCareSignal := machineWashSignal ||
		washTemp != nil ||
		negativeWash ||
		handWashOnly ||
		dryCleanOnly ||
		doNotDryClean ||
		positiveTumble ||
		negativeTumble ||
		positiveBleach ||
		negativeBleach ||
		positiveIron ||
		negativeIron

	result := ParseLabelResult{
		NameSuggestion:  "Care Label",
		FabricNotes:     fabricNotes,
		WashTempMax:     washTemp,
		MachineWashable: machineWashable,
		TumbleDry:       tumbleDry,
		DryCleanOnly:    dryCleanOnly && !doNotDryClean,
		IronAllowed:     ironAllowed,
		IronTemp:        ironTemp,
		BleachAllowed:   bleachAllowed,
		RawLabelText:    rawText,
	}

	return result, careRuleEvidence{
		HasCareSignal: hasCareSignal,
		Conflicting: (positiveTumble && negativeTumble) ||
			(positiveBleach && negativeBleach) ||
			(positiveIron && negativeIron) ||
			(dryCleanOnly && doNotDryClean) ||
			(machineWashSignal && negativeWash),
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

var washTempWithUnitPattern = regexp.MustCompile(`\b([1-9][0-9])\s*(?:c|ºc|celsius)\b`)
var washTempNearCarePattern = regexp.MustCompile(`\b(?:wash|lavar|lavado)\b.{0,16}\b(20|30|40|50|60|70|80|90|95)\b|\b(20|30|40|50|60|70|80|90|95)\b.{0,16}\b(?:wash|lavar|lavado)\b`)

func inferWashTemperature(normalized string) *int {
	if match := washTempWithUnitPattern.FindStringSubmatch(normalized); len(match) == 2 {
		if value, ok := parseWashTemp(match[1]); ok {
			return &value
		}
	}

	if match := washTempNearCarePattern.FindStringSubmatch(normalized); len(match) == 3 {
		candidate := match[1]
		if candidate == "" {
			candidate = match[2]
		}
		if value, ok := parseWashTemp(candidate); ok {
			return &value
		}
	}

	switch {
	case containsAny(normalized, "machine wash cold", "wash cold", "cold wash", "lavar en frio", "agua fria"):
		value := 30
		return &value
	case containsAny(normalized, "machine wash warm", "wash warm", "warm wash", "lavar tibio"):
		value := 40
		return &value
	case containsAny(normalized, "machine wash hot", "wash hot", "hot wash", "lavar caliente"):
		value := 60
		return &value
	default:
		return nil
	}
}

func parseWashTemp(value string) (int, bool) {
	temperature, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	if temperature < 20 || temperature > 95 {
		return 0, false
	}
	return temperature, true
}

func inferIronTemperature(normalized string) *string {
	switch {
	case containsAny(normalized, "cool iron", "low iron", "iron low", "planchar a baja", "baja temperatura"):
		value := "low"
		return &value
	case containsAny(normalized, "warm iron", "medium iron", "iron medium", "planchar a media", "temperatura media"):
		value := "medium"
		return &value
	case containsAny(normalized, "hot iron", "high iron", "iron high", "planchar a alta", "alta temperatura"):
		value := "high"
		return &value
	default:
		return nil
	}
}

var fabricAfterPercentPattern = regexp.MustCompile(`\b(\d{1,3})\s*%?\s*(cotton|polyester|elastane|linen|wool|viscose|nylon|acrylic|silk|spandex|algodon|poliester|lana|lino|seda|viscosa)\b`)
var fabricBeforePercentPattern = regexp.MustCompile(`\b(cotton|polyester|elastane|linen|wool|viscose|nylon|acrylic|silk|spandex|algodon|poliester|lana|lino|seda|viscosa)\s*(\d{1,3})\s*%`)

func extractFabricNotes(normalized string) []string {
	notes := make([]string, 0, 4)
	seen := map[string]bool{}

	for _, match := range fabricAfterPercentPattern.FindAllStringSubmatch(normalized, -1) {
		if len(match) != 3 {
			continue
		}
		addFabricNote(&notes, seen, match[1], match[2])
	}

	for _, match := range fabricBeforePercentPattern.FindAllStringSubmatchIndex(normalized, -1) {
		if len(match) != 6 {
			continue
		}
		prefixStart := match[0] - 3
		if prefixStart < 0 {
			prefixStart = 0
		}
		if strings.Contains(normalized[prefixStart:match[0]], "%") {
			continue
		}

		fabric := normalized[match[2]:match[3]]
		percent := normalized[match[4]:match[5]]
		addFabricNote(&notes, seen, percent, fabric)
	}

	return notes
}

func addFabricNote(notes *[]string, seen map[string]bool, percent string, fabric string) {
	value, err := strconv.Atoi(percent)
	if err != nil || value <= 0 || value > 100 {
		return
	}

	note := percent + "% " + normalizeFabricName(fabric)
	if seen[note] {
		return
	}
	seen[note] = true
	*notes = append(*notes, note)
}

func normalizeFabricName(fabric string) string {
	switch strings.TrimSpace(fabric) {
	case "algodon":
		return "cotton"
	case "poliester":
		return "polyester"
	case "lana":
		return "wool"
	case "lino":
		return "linen"
	case "seda":
		return "silk"
	case "viscosa":
		return "viscose"
	default:
		return strings.TrimSpace(fabric)
	}
}
