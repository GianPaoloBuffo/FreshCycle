package labelparser

import (
	"regexp"
	"strconv"
	"strings"
)

type careRuleEvidence struct {
	HasCareSignal   bool
	HasWashSignal   bool
	HasDrySignal    bool
	HasBleachSignal bool
	HasIronSignal   bool
	HasCleanSignal  bool
	Conflicting     bool
}

func parseCareLabelText(text string) (ParseLabelResult, careRuleEvidence) {
	rawText := normalizeOCRText(text)
	normalized := normalizeForRules(rawText)
	compact := compactForRules(normalized)

	washTemp := inferWashTemperature(normalized)
	negativeWash := containsAny(normalized, "do not wash", "dont wash", "no lavar") || containsAny(compact, "donotwash", "dontwash", "nolavar")
	handWashOnly := containsAny(normalized, "hand wash only", "hand wash", "lavar a mano", "lavado a mano")
	dryCleanOnly := containsAny(normalized, "dry clean only", "professional dry clean only", "limpieza en seco solamente", "solo limpieza en seco")
	doNotDryClean := containsAny(normalized, "do not dry clean", "dont dry clean", "no lavar en seco", "no limpieza en seco") || containsAny(compact, "donotdryclean", "dontdryclean", "nolavarseco")

	machineWashSignal := containsAny(normalized, "machine wash", "machine washable", "wash cold", "cold wash", "wash warm", "wash hot", "lavar a maquina", "lavado a maquina")
	machineWashable := !negativeWash && !dryCleanOnly && !handWashOnly && (machineWashSignal || washTemp != nil)

	negativeTumble := containsAny(normalized, "do not tumble dry", "dont tumble dry", "no tumble dry", "avoid tumble dry", "avoid tumble drying", "dry flat", "line dry", "lay flat", "no usar secadora") ||
		containsAny(compact, "donottumbledry", "donttumbledry", "notumbledry", "dryflat", "linedry", "layflat")
	positiveTumble := containsAny(normalized, "tumble dry", "secadora")
	tumbleDry := positiveTumble && !negativeTumble

	negativeBleach := containsAny(normalized, "do not bleach", "dont bleach", "no bleach", "no usar lejia", "no usar blanqueador", "no blanquear") ||
		containsBleachProhibition(normalized, compact)
	positiveBleach := containsAny(normalized, "bleach allowed", "bleach when needed", "non chlorine bleach", "non-chlorine bleach", "only non chlorine bleach", "blanqueador sin cloro")
	bleachAllowed := positiveBleach && !negativeBleach

	negativeIron := containsAny(normalized, "do not iron", "dont iron", "no iron", "no planchar") || containsAny(compact, "donotiron", "dontiron", "noiron", "noplanchar")
	positiveIron := containsIronSignal(normalized, compact)
	ironAllowed := positiveIron && !negativeIron
	ironTemp := inferIronTemperature(normalized, compact)
	if !ironAllowed {
		ironTemp = nil
	}

	fabricNotes := extractFabricNotes(normalized)
	hasWashSignal := machineWashSignal || washTemp != nil || negativeWash || handWashOnly
	hasDrySignal := positiveTumble || negativeTumble
	hasBleachSignal := positiveBleach || negativeBleach
	hasIronSignal := positiveIron || negativeIron
	hasCleanSignal := dryCleanOnly || doNotDryClean
	hasCareSignal := hasWashSignal ||
		hasDrySignal ||
		hasBleachSignal ||
		hasIronSignal ||
		hasCleanSignal ||
		washTemp != nil ||
		negativeWash ||
		handWashOnly

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
		HasCareSignal:   hasCareSignal,
		HasWashSignal:   hasWashSignal,
		HasDrySignal:    hasDrySignal,
		HasBleachSignal: hasBleachSignal,
		HasIronSignal:   hasIronSignal,
		HasCleanSignal:  hasCleanSignal,
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

func compactForRules(value string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		default:
			return -1
		}
	}, value)
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
	case machineWashColdLooksGarbled(normalized):
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

func machineWashColdLooksGarbled(normalized string) bool {
	if !strings.Contains(normalized, "machine wash") {
		return false
	}

	index := strings.Index(normalized, "machine wash")
	after := normalized[index:]
	if len(after) > 32 {
		after = after[:32]
	}

	return containsAny(after, "coid", "c0ld", "co ld", "oo es", "o o es")
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

func containsBleachProhibition(normalized string, compact string) bool {
	if containsAny(compact, "donotbleach", "dontbleach", "nobleach", "nousarlejia", "nousarblanqueador", "noblanquear") {
		return true
	}

	return regexp.MustCompile(`do\s*not\s*blea(?:ch|c|o|q)?`).MatchString(normalized) ||
		regexp.MustCompile(`dono?t?blea(?:ch|c|o|q)?`).MatchString(compact)
}

func containsIronSignal(normalized string, compact string) bool {
	if containsAny(normalized, "iron", "cool iron", "warm iron", "hot iron", "planchar") {
		return true
	}
	if containsAny(compact, "iron", "planchar") {
		return true
	}

	return regexp.MustCompile(`\b(?:a\s+)?(?:i\s*)?r[0o]n\s+l[0o][nw]\s+heat\b`).MatchString(normalized)
}

func inferIronTemperature(normalized string, compact string) *string {
	switch {
	case containsAny(normalized, "cool iron", "low iron", "iron low", "low heat", "planchar a baja", "baja temperatura") ||
		containsAny(compact, "cooliron", "lowiron", "ironlow", "lowheat", "lonheat", "l0wheat"):
		value := "low"
		return &value
	case containsAny(normalized, "warm iron", "medium iron", "iron medium", "medium heat", "planchar a media", "temperatura media") ||
		containsAny(compact, "warmiron", "mediumiron", "ironmedium", "mediumheat"):
		value := "medium"
		return &value
	case containsAny(normalized, "hot iron", "high iron", "iron high", "high heat", "planchar a alta", "alta temperatura") ||
		containsAny(compact, "hotiron", "highiron", "ironhigh", "highheat"):
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
