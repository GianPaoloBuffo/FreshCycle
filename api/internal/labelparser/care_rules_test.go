package labelparser

import "testing"

func TestParseCareLabelTextCommonEnglishLabel(t *testing.T) {
	result, evidence := parseCareLabelText("100% cotton. Machine wash cold. Do not bleach. Tumble dry low. Cool iron if needed.")

	if !evidence.HasCareSignal || evidence.Conflicting {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
	if result.WashTempMax == nil || *result.WashTempMax != 30 {
		t.Fatalf("expected cold wash to infer 30C, got %#v", result.WashTempMax)
	}
	if !result.MachineWashable {
		t.Fatal("expected machine washable")
	}
	if !result.TumbleDry {
		t.Fatal("expected tumble dry allowed")
	}
	if result.BleachAllowed {
		t.Fatal("expected bleach to be disallowed")
	}
	if !result.IronAllowed || result.IronTemp == nil || *result.IronTemp != "low" {
		t.Fatalf("expected low iron, got allowed=%v temp=%#v", result.IronAllowed, result.IronTemp)
	}
	if len(result.FabricNotes) != 1 || result.FabricNotes[0] != "100% cotton" {
		t.Fatalf("expected cotton fabric note, got %#v", result.FabricNotes)
	}
}

func TestParseCareLabelTextTemperatureAndFabricPercentages(t *testing.T) {
	result, _ := parseCareLabelText("55% linen 45% cotton. Wash at 40C. Do not tumble dry. Warm iron.")

	if result.WashTempMax == nil || *result.WashTempMax != 40 {
		t.Fatalf("expected 40C wash, got %#v", result.WashTempMax)
	}
	if result.TumbleDry {
		t.Fatal("expected tumble dry to be disallowed")
	}
	if !result.IronAllowed || result.IronTemp == nil || *result.IronTemp != "medium" {
		t.Fatalf("expected medium iron, got allowed=%v temp=%#v", result.IronAllowed, result.IronTemp)
	}
	if len(result.FabricNotes) != 2 {
		t.Fatalf("expected two fabric notes, got %#v", result.FabricNotes)
	}
}

func TestParseCareLabelTextDryCleanOnly(t *testing.T) {
	result, _ := parseCareLabelText("Dry clean only. Do not iron. Do not bleach.")

	if !result.DryCleanOnly {
		t.Fatal("expected dry clean only")
	}
	if result.MachineWashable {
		t.Fatal("did not expect machine washable")
	}
	if result.IronAllowed || result.IronTemp != nil {
		t.Fatalf("expected iron disallowed, got allowed=%v temp=%#v", result.IronAllowed, result.IronTemp)
	}
}

func TestParseCareLabelTextSpanishLabel(t *testing.T) {
	result, _ := parseCareLabelText("100% algodón. Lavar a máquina 30ºC. No usar lejía. Planchar a baja temperatura.")

	if result.WashTempMax == nil || *result.WashTempMax != 30 {
		t.Fatalf("expected 30C wash, got %#v", result.WashTempMax)
	}
	if !result.MachineWashable {
		t.Fatal("expected machine washable")
	}
	if result.BleachAllowed {
		t.Fatal("expected bleach disallowed")
	}
	if !result.IronAllowed || result.IronTemp == nil || *result.IronTemp != "low" {
		t.Fatalf("expected low iron, got allowed=%v temp=%#v", result.IronAllowed, result.IronTemp)
	}
	if len(result.FabricNotes) != 1 || result.FabricNotes[0] != "100% cotton" {
		t.Fatalf("expected normalized Spanish cotton note, got %#v", result.FabricNotes)
	}
}

func TestParseCareLabelTextDetectsConflictingRules(t *testing.T) {
	_, evidence := parseCareLabelText("Bleach when needed. Do not bleach.")

	if !evidence.Conflicting {
		t.Fatal("expected conflicting bleach rules")
	}
}
