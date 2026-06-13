package labelparser

import (
	"fmt"
	"strings"
)

const (
	SymbolWashTub             = "wash_tub"
	SymbolWashDoNotWash       = "wash_do_not_wash"
	SymbolWashHand            = "wash_hand"
	SymbolWashGentle          = "wash_gentle"
	SymbolWashVeryGentle      = "wash_very_gentle"
	SymbolBleachAllowed       = "bleach_allowed"
	SymbolBleachNonChlorine   = "bleach_non_chlorine"
	SymbolBleachDoNotBleach   = "bleach_do_not_bleach"
	SymbolDryTumble           = "dry_tumble"
	SymbolDryTumbleLow        = "dry_tumble_low"
	SymbolDryTumbleMedium     = "dry_tumble_medium"
	SymbolDryTumbleHigh       = "dry_tumble_high"
	SymbolDryDoNotTumble      = "dry_do_not_tumble"
	SymbolDryLine             = "dry_line"
	SymbolDryFlat             = "dry_flat"
	SymbolDryDrip             = "dry_drip"
	SymbolDryShade            = "dry_shade"
	SymbolIronAllowed         = "iron_allowed"
	SymbolIronLow             = "iron_low"
	SymbolIronMedium          = "iron_medium"
	SymbolIronHigh            = "iron_high"
	SymbolIronDoNotIron       = "iron_do_not_iron"
	SymbolDryClean            = "professional_dry_clean"
	SymbolDryCleanP           = "professional_dry_clean_p"
	SymbolDryCleanF           = "professional_dry_clean_f"
	SymbolDryCleanDoNot       = "professional_do_not_dry_clean"
	SymbolWetClean            = "professional_wet_clean"
	SymbolWetCleanDoNot       = "professional_do_not_wet_clean"
	SymbolModifierCross       = "modifier_cross"
	SymbolModifierOneDot      = "modifier_one_dot"
	SymbolModifierTwoDots     = "modifier_two_dots"
	SymbolModifierThreeDots   = "modifier_three_dots"
	SymbolModifierOneBar      = "modifier_one_bar"
	SymbolModifierTwoBars     = "modifier_two_bars"
	SymbolModifierLetterP     = "modifier_letter_p"
	SymbolModifierLetterF     = "modifier_letter_f"
	SymbolModifierLetterW     = "modifier_letter_w"
	SymbolModifierTemperature = "modifier_temperature"
)

type LaundrySymbolDefinition struct {
	Class       string
	Label       string
	CareField   string
	ImpliedText string
}

var laundrySymbolDefinitions = []LaundrySymbolDefinition{
	{Class: SymbolWashTub, Label: "wash tub", CareField: "wash", ImpliedText: "machine wash"},
	{Class: SymbolWashDoNotWash, Label: "do not wash", CareField: "wash", ImpliedText: "do not wash"},
	{Class: SymbolWashHand, Label: "hand wash", CareField: "wash", ImpliedText: "hand wash"},
	{Class: SymbolWashGentle, Label: "gentle wash cycle", CareField: "wash", ImpliedText: "delicate cycle"},
	{Class: SymbolWashVeryGentle, Label: "very gentle wash cycle", CareField: "wash", ImpliedText: "delicate cycle"},
	{Class: SymbolBleachAllowed, Label: "bleach allowed", CareField: "bleach", ImpliedText: "bleach allowed"},
	{Class: SymbolBleachNonChlorine, Label: "non-chlorine bleach", CareField: "bleach", ImpliedText: "non chlorine bleach"},
	{Class: SymbolBleachDoNotBleach, Label: "do not bleach", CareField: "bleach", ImpliedText: "do not bleach"},
	{Class: SymbolDryTumble, Label: "tumble dry", CareField: "drying", ImpliedText: "tumble dry"},
	{Class: SymbolDryTumbleLow, Label: "tumble dry low", CareField: "drying", ImpliedText: "tumble dry low"},
	{Class: SymbolDryTumbleMedium, Label: "tumble dry medium", CareField: "drying", ImpliedText: "tumble dry medium"},
	{Class: SymbolDryTumbleHigh, Label: "tumble dry high", CareField: "drying", ImpliedText: "tumble dry high"},
	{Class: SymbolDryDoNotTumble, Label: "do not tumble dry", CareField: "drying", ImpliedText: "do not tumble dry"},
	{Class: SymbolDryLine, Label: "line dry", CareField: "drying", ImpliedText: "line dry"},
	{Class: SymbolDryFlat, Label: "dry flat", CareField: "drying", ImpliedText: "dry flat"},
	{Class: SymbolDryDrip, Label: "drip dry", CareField: "drying", ImpliedText: "line dry"},
	{Class: SymbolDryShade, Label: "dry in shade", CareField: "drying", ImpliedText: "line dry"},
	{Class: SymbolIronAllowed, Label: "iron allowed", CareField: "ironing", ImpliedText: "iron"},
	{Class: SymbolIronLow, Label: "iron low", CareField: "ironing", ImpliedText: "cool iron"},
	{Class: SymbolIronMedium, Label: "iron medium", CareField: "ironing", ImpliedText: "warm iron"},
	{Class: SymbolIronHigh, Label: "iron high", CareField: "ironing", ImpliedText: "hot iron"},
	{Class: SymbolIronDoNotIron, Label: "do not iron", CareField: "ironing", ImpliedText: "do not iron"},
	{Class: SymbolDryClean, Label: "professional dry clean", CareField: "professional_cleaning", ImpliedText: "dry clean"},
	{Class: SymbolDryCleanP, Label: "dry clean P", CareField: "professional_cleaning", ImpliedText: "dry clean"},
	{Class: SymbolDryCleanF, Label: "dry clean F", CareField: "professional_cleaning", ImpliedText: "dry clean"},
	{Class: SymbolDryCleanDoNot, Label: "do not dry clean", CareField: "professional_cleaning", ImpliedText: "do not dry clean"},
	{Class: SymbolWetClean, Label: "professional wet clean", CareField: "professional_cleaning", ImpliedText: "wet clean"},
	{Class: SymbolWetCleanDoNot, Label: "do not wet clean", CareField: "professional_cleaning", ImpliedText: "do not wet clean"},
}

var laundrySymbolDefinitionByClass = buildLaundrySymbolDefinitionByClass()

func buildLaundrySymbolDefinitionByClass() map[string]LaundrySymbolDefinition {
	definitions := make(map[string]LaundrySymbolDefinition, len(laundrySymbolDefinitions))
	for _, definition := range laundrySymbolDefinitions {
		definitions[definition.Class] = definition
	}
	return definitions
}

func NormalizeLaundrySymbolClass(value string) (string, bool) {
	normalized := normalizeSymbolName(value)
	if normalized == "" {
		return "", false
	}

	if definition, ok := laundrySymbolDefinitionByClass[normalized]; ok {
		return definition.Class, true
	}
	if strings.HasPrefix(normalized, "wash_temperature_") {
		return normalized, true
	}

	compact := compactForRules(normalized)
	switch {
	case containsAny(normalized, "do not wash", "dont wash") || containsAny(compact, "donotwash", "dontwash"):
		return SymbolWashDoNotWash, true
	case containsAny(normalized, "hand wash"):
		return SymbolWashHand, true
	case containsAny(normalized, "wash", "tub") && inferWashTemperature(normalized) != nil:
		return normalizeWashTemperatureClass(normalized)
	case containsAny(normalized, "wash") && containsAny(normalized, "very gentle", "double bar"):
		return SymbolWashVeryGentle, true
	case containsAny(normalized, "wash") && containsAny(normalized, "gentle", "underline", "one bar"):
		return SymbolWashGentle, true
	case containsAny(normalized, "wash", "tub"):
		return normalizeWashTemperatureClass(normalized)
	case containsAny(normalized, "do not bleach", "dont bleach", "no bleach") || containsAny(compact, "donotbleach", "dontbleach", "nobleach") ||
		(containsAny(normalized, "cross", "crossed") && containsAny(normalized, "triangle", "bleach")):
		return SymbolBleachDoNotBleach, true
	case containsAny(normalized, "non chlorine bleach", "oxygen bleach"):
		return SymbolBleachNonChlorine, true
	case containsAny(normalized, "bleach", "triangle"):
		return SymbolBleachAllowed, true
	case containsAny(normalized, "do not tumble dry", "dont tumble dry", "no tumble dry") || containsAny(compact, "donottumbledry", "donttumbledry", "notumbledry"):
		return SymbolDryDoNotTumble, true
	case containsAny(normalized, "tumble dry low", "low tumble", "one dot tumble"):
		return SymbolDryTumbleLow, true
	case containsAny(normalized, "tumble dry medium", "normal tumble", "two dot tumble"):
		return SymbolDryTumbleMedium, true
	case containsAny(normalized, "tumble dry high", "three dot tumble"):
		return SymbolDryTumbleHigh, true
	case containsAny(normalized, "tumble dry", "square circle"):
		return SymbolDryTumble, true
	case containsAny(normalized, "dry flat", "flat dry"):
		return SymbolDryFlat, true
	case containsAny(normalized, "line dry", "hang dry"):
		return SymbolDryLine, true
	case containsAny(normalized, "drip dry"):
		return SymbolDryDrip, true
	case containsAny(normalized, "shade"):
		return SymbolDryShade, true
	case containsAny(normalized, "do not iron", "dont iron", "no iron") || containsAny(compact, "donotiron", "dontiron", "noiron"):
		return SymbolIronDoNotIron, true
	case containsAny(normalized, "iron low", "cool iron", "one dot iron"):
		return SymbolIronLow, true
	case containsAny(normalized, "iron medium", "warm iron", "two dot iron"):
		return SymbolIronMedium, true
	case containsAny(normalized, "iron high", "hot iron", "three dot iron"):
		return SymbolIronHigh, true
	case containsAny(normalized, "iron"):
		return SymbolIronAllowed, true
	case containsAny(normalized, "do not dry clean", "dont dry clean", "no dry clean") || containsAny(compact, "donotdryclean", "dontdryclean", "nodryclean"):
		return SymbolDryCleanDoNot, true
	case containsAny(normalized, "dry clean p", "letter p") || (containsAny(normalized, "circle p", "p circle") && containsAny(normalized, "clean")):
		return SymbolDryCleanP, true
	case containsAny(normalized, "dry clean f", "letter f"):
		return SymbolDryCleanF, true
	case containsAny(normalized, "dry clean", "professional clean"):
		return SymbolDryClean, true
	case containsAny(normalized, "do not wet clean", "no wet clean"):
		return SymbolWetCleanDoNot, true
	case containsAny(normalized, "wet clean"):
		return SymbolWetClean, true
	case containsAny(normalized, "cross", "crossed out"):
		return SymbolModifierCross, true
	case containsAny(normalized, "one dot", "1 dot"):
		return SymbolModifierOneDot, true
	case containsAny(normalized, "two dots", "2 dots"):
		return SymbolModifierTwoDots, true
	case containsAny(normalized, "three dots", "3 dots"):
		return SymbolModifierThreeDots, true
	case containsAny(normalized, "one bar", "underline"):
		return SymbolModifierOneBar, true
	case containsAny(normalized, "two bars", "double underline"):
		return SymbolModifierTwoBars, true
	default:
		return normalized, false
	}
}

func normalizeWashTemperatureClass(normalized string) (string, bool) {
	if temperature := inferWashTemperature(normalized); temperature != nil {
		return fmt.Sprintf("wash_temperature_%d", *temperature), true
	}
	return SymbolWashTub, true
}

func normalizeSymbolName(value string) string {
	value = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(value)
	return strings.TrimSpace(normalizeForRules(value))
}

func laundrySymbolLabel(class string, fallback string) string {
	if definition, ok := laundrySymbolDefinitionByClass[class]; ok {
		return definition.Label
	}
	if strings.HasPrefix(class, "wash_temperature_") {
		return strings.TrimPrefix(class, "wash_temperature_") + "C wash temperature"
	}
	return strings.TrimSpace(fallback)
}

func laundrySymbolImpliedText(class string) string {
	if definition, ok := laundrySymbolDefinitionByClass[class]; ok {
		return definition.ImpliedText
	}
	if strings.HasPrefix(class, "wash_temperature_") {
		return "wash " + strings.TrimPrefix(class, "wash_temperature_") + " c"
	}
	return ""
}
