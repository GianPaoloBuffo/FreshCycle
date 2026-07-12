package labelparser

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	washStatusUnknown      = "unknown"
	washStatusMachine      = "machine_wash"
	washStatusHand         = "hand_wash"
	washStatusDoNotWash    = "do_not_wash"
	washStatusDryCleanOnly = "dry_clean_only"

	bleachStatusUnknown         = "unknown"
	bleachStatusAllowed         = "allowed"
	bleachStatusNonChlorineOnly = "non_chlorine_only"
	bleachStatusDoNotBleach     = "do_not_bleach"

	dryingStatusUnknown        = "unknown"
	dryingStatusTumbleDry      = "tumble_dry"
	dryingStatusDoNotTumbleDry = "do_not_tumble_dry"
	dryingStatusLineDry        = "line_dry"
	dryingStatusDryFlat        = "dry_flat"

	ironingStatusUnknown   = "unknown"
	ironingStatusAllowed   = "iron_allowed"
	ironingStatusDoNotIron = "do_not_iron"

	cleaningStatusUnknown       = "unknown"
	cleaningStatusDryCleanOnly  = "dry_clean_only"
	cleaningStatusDryClean      = "dry_clean_allowed"
	cleaningStatusDoNotDryClean = "do_not_dry_clean"
)

func (o *ClientOCR) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		o.Text = text
		return nil
	}

	var payload struct {
		Text       string   `json:"text"`
		RawText    string   `json:"raw_text"`
		Confidence *float64 `json:"confidence"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	o.Text = payload.Text
	if strings.TrimSpace(o.Text) == "" {
		o.Text = payload.RawText
	}
	o.Confidence = payload.Confidence
	return nil
}

func (s *ClientSymbol) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err == nil {
		s.Name = name
		s.Class, _ = NormalizeLaundrySymbolClass(name)
		s.Label = laundrySymbolLabel(s.Class, name)
		return nil
	}

	var payload struct {
		Name        string             `json:"name"`
		Symbol      string             `json:"symbol"`
		Label       string             `json:"label"`
		Type        string             `json:"type"`
		Class       string             `json:"class"`
		Confidence  *float64           `json:"confidence"`
		Box         *SymbolBoundingBox `json:"box"`
		BoundingBox *SymbolBoundingBox `json:"bounding_box"`
		Frame       *SymbolBoundingBox `json:"frame"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	for _, candidate := range []string{payload.Name, payload.Symbol, payload.Label, payload.Type, payload.Class} {
		if strings.TrimSpace(candidate) != "" {
			s.Name = candidate
			break
		}
	}
	s.Class, _ = NormalizeLaundrySymbolClass(firstNonEmpty(payload.Class, payload.Symbol, payload.Label, payload.Type, payload.Name))
	s.Label = firstNonEmpty(payload.Label, laundrySymbolLabel(s.Class, s.Name))
	s.Confidence = payload.Confidence
	s.Box = firstBox(payload.Box, payload.BoundingBox, payload.Frame)
	return nil
}

func ScanLabel(ctx context.Context, parser Parser, input ScanLabelInput) (ScanLabelResult, error) {
	if parser == nil {
		return ScanLabelResult{}, ErrProviderUnavailable
	}

	if scanner, ok := parser.(Scanner); ok {
		result, err := scanner.ScanLabel(ctx, input)
		if err != nil {
			return ScanLabelResult{}, err
		}
		return normalizeScanLabelResult(result), nil
	}

	result, err := parser.ParseLabel(ctx, input.ParseLabelInput)
	if err != nil {
		return ScanLabelResult{}, err
	}

	return scanFromParseLabelResult(result, careRuleEvidence{}, "legacy parser", 0), nil
}

func decodeScanLabelProviderOutput(output string) (ScanLabelResult, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return ScanLabelResult{}, fmt.Errorf("%w: missing scan-label output", ErrUpstreamParseRejected)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		return ScanLabelResult{}, fmt.Errorf("decode scan-label structured output: %w", err)
	}

	if _, ok := raw["name_suggestion"]; ok {
		var legacy ParseLabelResult
		if err := json.Unmarshal([]byte(output), &legacy); err != nil {
			return ScanLabelResult{}, fmt.Errorf("decode legacy structured output: %w", err)
		}
		result := scanFromParseLabelResult(legacy, careRuleEvidence{}, "multimodal provider", 0.62)
		result.PaidFallbackUsed = true
		return result, nil
	}

	var result ScanLabelResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return ScanLabelResult{}, fmt.Errorf("decode scan-label structured output: %w", err)
	}

	if result.Provider == "" {
		result.Provider = "multimodal_fallback"
	}
	if result.Route == "" {
		result.Route = "multimodal_fallback"
	}
	result.PaidFallbackUsed = true
	return normalizeScanLabelResult(result), nil
}

func scanFromClientEvidence(input ScanLabelInput) (ScanLabelResult, bool) {
	parts := make([]string, 0, 2)
	if input.ClientOCR != nil && strings.TrimSpace(input.ClientOCR.Text) != "" {
		parts = append(parts, input.ClientOCR.Text)
	}
	if detectedText := symbolDetectionsToCareText(input.DetectedSymbols); detectedText != "" {
		parts = append(parts, detectedText)
	}
	if symbolText := clientSymbolsToCareText(input.ClientSymbols); symbolText != "" {
		parts = append(parts, symbolText)
	}
	if len(parts) == 0 {
		return ScanLabelResult{}, false
	}

	rawText := normalizeOCRText(strings.Join(parts, " "))
	legacy, evidence := parseCareLabelText(rawText)
	if !evidence.HasCareSignal {
		return ScanLabelResult{}, false
	}

	confidence := clientEvidenceConfidence(input, evidence)
	result := scanFromParseLabelResult(legacy, evidence, "client OCR and symbol hints", confidence)
	result.Explanation = "FreshCycle inferred care instructions from client-provided OCR and symbol hints."
	result.SymbolDetections = normalizeSymbolDetections(input.DetectedSymbols)
	result.Provider = "local_rules"
	result.Route = "local_rules"
	return result, true
}

func scanFromOCRDetails(details OCRParseDetails) ScanLabelResult {
	confidence := details.AverageConfidence / 100
	if details.WordCount == 0 {
		confidence = 0.3
	}
	if len(details.FallbackReasons) > 0 && confidence > 0.64 {
		confidence = 0.64
	}

	result := scanFromParseLabelResult(details.Result, details.Evidence, "server OCR", confidence)
	if len(details.FallbackReasons) > 0 {
		result.Explanation = "FreshCycle parsed the label with OCR, but the scan needs review because " + strings.Join(details.FallbackReasons, ", ") + "."
	}
	return result
}

func scanFromParseLabelResult(result ParseLabelResult, evidence careRuleEvidence, source string, confidence float64) ScanLabelResult {
	rawText := normalizeOCRText(result.RawLabelText)
	normalized := normalizeForRules(rawText)
	compact := compactForRules(normalized)

	if !evidence.HasCareSignal && rawText != "" {
		_, parsedEvidence := parseCareLabelText(rawText)
		evidence = mergeCareEvidence(evidence, parsedEvidence)
	}
	evidence = mergeCareEvidence(evidence, evidenceFromParseLabelResult(result))

	washStatus := inferScanWashStatus(result, normalized, compact)
	washCycle := inferWashCycle(normalized)
	bleachStatus, bleachKind := inferScanBleachStatus(result, evidence, normalized, compact)
	dryingStatus, dryingTemperature := inferScanDryingStatus(result, evidence, normalized, compact)
	ironingStatus := inferScanIroningStatus(result, evidence, normalized, compact)
	cleaningStatus, cleaningMethod := inferScanCleaningStatus(result, evidence, normalized, compact)

	scan := ScanLabelResult{
		Wash: WashInstruction{
			Status:          washStatus,
			MaxTemperatureC: result.WashTempMax,
			Cycle:           washCycle,
		},
		Bleach: BleachInstruction{
			Status: bleachStatus,
			Kind:   bleachKind,
		},
		Drying: DryingInstruction{
			Status:      dryingStatus,
			Temperature: dryingTemperature,
		},
		Ironing: IroningInstruction{
			Status:      ironingStatus,
			Temperature: result.IronTemp,
		},
		ProfessionalCleaning: ProfessionalCleaningInstruction{
			Status: cleaningStatus,
			Method: cleaningMethod,
		},
		RawText:    rawText,
		Confidence: confidence,
		Provider:   providerFromSource(source),
		Route:      routeFromSource(source),
	}

	scan.Wash.Summary = washSummary(scan.Wash)
	scan.Bleach.Summary = bleachSummary(scan.Bleach)
	scan.Drying.Summary = dryingSummary(scan.Drying)
	scan.Ironing.Summary = ironingSummary(scan.Ironing)
	scan.ProfessionalCleaning.Summary = cleaningSummary(scan.ProfessionalCleaning)
	scan.Explanation = defaultScanExplanation(source, scan.Confidence)

	return normalizeScanLabelResult(scan)
}

func providerFromSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.Contains(source, "client") || strings.Contains(source, "rules"):
		return "local_rules"
	case strings.Contains(source, "ocr"):
		return "server_ocr"
	case strings.Contains(source, "fallback") || strings.Contains(source, "multimodal") || strings.Contains(source, "provider"):
		return "multimodal_fallback"
	case strings.Contains(source, "stub"):
		return "stub"
	default:
		return ""
	}
}

func routeFromSource(source string) string {
	provider := providerFromSource(source)
	if provider != "" {
		return provider
	}
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(source, " ", "_")))
}

func normalizeScanLabelResult(result ScanLabelResult) ScanLabelResult {
	result.Wash.Status = validStatus(result.Wash.Status, []string{washStatusMachine, washStatusHand, washStatusDoNotWash, washStatusDryCleanOnly, washStatusUnknown})
	result.Bleach.Status = validStatus(result.Bleach.Status, []string{bleachStatusAllowed, bleachStatusNonChlorineOnly, bleachStatusDoNotBleach, bleachStatusUnknown})
	result.Drying.Status = validStatus(result.Drying.Status, []string{dryingStatusTumbleDry, dryingStatusDoNotTumbleDry, dryingStatusLineDry, dryingStatusDryFlat, dryingStatusUnknown})
	result.Ironing.Status = validStatus(result.Ironing.Status, []string{ironingStatusAllowed, ironingStatusDoNotIron, ironingStatusUnknown})
	result.ProfessionalCleaning.Status = validStatus(result.ProfessionalCleaning.Status, []string{cleaningStatusDryCleanOnly, cleaningStatusDryClean, cleaningStatusDoNotDryClean, cleaningStatusUnknown})

	result.RawText = normalizeOCRText(result.RawText)
	if result.Wash.Summary == "" {
		result.Wash.Summary = washSummary(result.Wash)
	}
	if result.Bleach.Summary == "" {
		result.Bleach.Summary = bleachSummary(result.Bleach)
	}
	if result.Drying.Summary == "" {
		result.Drying.Summary = dryingSummary(result.Drying)
	}
	if result.Ironing.Summary == "" {
		result.Ironing.Summary = ironingSummary(result.Ironing)
	}
	if result.ProfessionalCleaning.Summary == "" {
		result.ProfessionalCleaning.Summary = cleaningSummary(result.ProfessionalCleaning)
	}

	result.Confidence = normalizeConfidence(result.Confidence, defaultConfidenceForScan(result))
	if strings.TrimSpace(result.Explanation) == "" {
		result.Explanation = defaultScanExplanation("label parser", result.Confidence)
	}
	result.UncertainFields = mergeUniqueStrings(result.UncertainFields, defaultUncertainFields(result))
	fillInstructionMetadata(&result)
	result.SymbolDetections = normalizeSymbolDetections(result.SymbolDetections)
	result.NeedsUserConfirmation = result.NeedsUserConfirmation || result.Confidence < 0.75 || len(result.UncertainFields) > 0

	if result.UncertainFields == nil {
		result.UncertainFields = []string{}
	}
	if result.RoutingReasons == nil {
		result.RoutingReasons = []string{}
	}

	return result
}

func fillInstructionMetadata(result *ScanLabelResult) {
	result.Wash.Confidence = normalizeConfidence(result.Wash.Confidence, defaultFieldConfidence(result.Confidence, result.Wash.Status, result.UncertainFields, "wash"))
	result.Bleach.Confidence = normalizeConfidence(result.Bleach.Confidence, defaultFieldConfidence(result.Confidence, result.Bleach.Status, result.UncertainFields, "bleach"))
	result.Drying.Confidence = normalizeConfidence(result.Drying.Confidence, defaultFieldConfidence(result.Confidence, result.Drying.Status, result.UncertainFields, "drying"))
	result.Ironing.Confidence = normalizeConfidence(result.Ironing.Confidence, defaultFieldConfidence(result.Confidence, result.Ironing.Status, result.UncertainFields, "ironing"))
	result.ProfessionalCleaning.Confidence = normalizeConfidence(result.ProfessionalCleaning.Confidence, defaultFieldConfidence(result.Confidence, result.ProfessionalCleaning.Status, result.UncertainFields, "professional_cleaning"))

	if strings.TrimSpace(result.Wash.Explanation) == "" {
		result.Wash.Explanation = instructionExplanation("wash", result.Wash.Summary, result.Wash.Confidence)
	}
	if strings.TrimSpace(result.Bleach.Explanation) == "" {
		result.Bleach.Explanation = instructionExplanation("bleach", result.Bleach.Summary, result.Bleach.Confidence)
	}
	if strings.TrimSpace(result.Drying.Explanation) == "" {
		result.Drying.Explanation = instructionExplanation("drying", result.Drying.Summary, result.Drying.Confidence)
	}
	if strings.TrimSpace(result.Ironing.Explanation) == "" {
		result.Ironing.Explanation = instructionExplanation("ironing", result.Ironing.Summary, result.Ironing.Confidence)
	}
	if strings.TrimSpace(result.ProfessionalCleaning.Explanation) == "" {
		result.ProfessionalCleaning.Explanation = instructionExplanation("professional cleaning", result.ProfessionalCleaning.Summary, result.ProfessionalCleaning.Confidence)
	}

	result.Wash.NeedsConfirmation = result.Wash.NeedsConfirmation || fieldNeedsConfirmation(result.Wash.Confidence, result.Wash.Status, result.UncertainFields, "wash")
	result.Bleach.NeedsConfirmation = result.Bleach.NeedsConfirmation || fieldNeedsConfirmation(result.Bleach.Confidence, result.Bleach.Status, result.UncertainFields, "bleach")
	result.Drying.NeedsConfirmation = result.Drying.NeedsConfirmation || fieldNeedsConfirmation(result.Drying.Confidence, result.Drying.Status, result.UncertainFields, "drying")
	result.Ironing.NeedsConfirmation = result.Ironing.NeedsConfirmation || fieldNeedsConfirmation(result.Ironing.Confidence, result.Ironing.Status, result.UncertainFields, "ironing")
	result.ProfessionalCleaning.NeedsConfirmation = result.ProfessionalCleaning.NeedsConfirmation || fieldNeedsConfirmation(result.ProfessionalCleaning.Confidence, result.ProfessionalCleaning.Status, result.UncertainFields, "professional_cleaning")
}

func defaultFieldConfidence(rootConfidence float64, status string, uncertainFields []string, field string) float64 {
	confidence := rootConfidence
	if confidence <= 0 {
		confidence = 0.62
	}
	if status == "unknown" {
		if confidence > 0.38 {
			confidence = 0.38
		}
	}
	if fieldIsUncertain(uncertainFields, field) && confidence > 0.64 {
		confidence = 0.64
	}
	return confidence
}

func fieldNeedsConfirmation(confidence float64, status string, uncertainFields []string, field string) bool {
	return status == "unknown" || confidence < 0.72 || fieldIsUncertain(uncertainFields, field)
}

func fieldIsUncertain(uncertainFields []string, field string) bool {
	for _, uncertainField := range uncertainFields {
		if uncertainField == field || strings.HasPrefix(uncertainField, field+".") {
			return true
		}
	}
	return false
}

func instructionExplanation(field string, summary string, confidence float64) string {
	if confidence < 0.55 {
		return "FreshCycle could not confidently determine the " + field + " instruction."
	}
	return summary
}

func shouldAcceptLocalScan(result ScanLabelResult, minConfidence float64, minKnownFields int) bool {
	result = normalizeScanLabelResult(result)
	if result.Confidence < minConfidence {
		return false
	}
	return knownInstructionCount(result) >= minKnownFields
}

func knownInstructionCount(result ScanLabelResult) int {
	count := 0
	if result.Wash.Status != washStatusUnknown {
		count++
	}
	if result.Bleach.Status != bleachStatusUnknown {
		count++
	}
	if result.Drying.Status != dryingStatusUnknown {
		count++
	}
	if result.Ironing.Status != ironingStatusUnknown {
		count++
	}
	if result.ProfessionalCleaning.Status != cleaningStatusUnknown {
		count++
	}
	return count
}

func inferScanWashStatus(result ParseLabelResult, normalized string, compact string) string {
	switch {
	case result.DryCleanOnly:
		return washStatusDryCleanOnly
	case containsAny(normalized, "do not wash", "dont wash", "no lavar") || containsAny(compact, "donotwash", "dontwash", "nolavar"):
		return washStatusDoNotWash
	case containsAny(normalized, "hand wash only", "hand wash", "lavar a mano", "lavado a mano"):
		return washStatusHand
	case result.MachineWashable || result.WashTempMax != nil || containsAny(normalized, "machine wash", "machine washable", "lavar a maquina", "lavado a maquina"):
		return washStatusMachine
	default:
		return washStatusUnknown
	}
}

func inferWashCycle(normalized string) *string {
	switch {
	case containsAny(normalized, "delicate cycle", "gentle cycle", "delicates", "lavado delicado"):
		return stringPtr("delicate")
	case containsAny(normalized, "permanent press"):
		return stringPtr("permanent_press")
	case containsAny(normalized, "normal cycle"):
		return stringPtr("normal")
	default:
		return nil
	}
}

func inferScanBleachStatus(result ParseLabelResult, evidence careRuleEvidence, normalized string, compact string) (string, *string) {
	negativeBleach := containsAny(normalized, "do not bleach", "dont bleach", "no bleach", "no blanquear") || containsBleachProhibition(normalized, compact)
	nonChlorineBleach := containsAny(normalized, "non chlorine bleach", "non-chlorine bleach", "sin cloro") || containsAny(compact, "nonchlorinebleach")
	positiveBleach := containsAny(normalized, "bleach allowed", "bleach when needed", "blanqueador sin cloro") || nonChlorineBleach

	switch {
	case negativeBleach:
		return bleachStatusDoNotBleach, nil
	case result.BleachAllowed || positiveBleach:
		if nonChlorineBleach {
			return bleachStatusNonChlorineOnly, stringPtr("non_chlorine")
		}
		return bleachStatusAllowed, stringPtr("any")
	case evidence.HasBleachSignal:
		return bleachStatusDoNotBleach, nil
	default:
		return bleachStatusUnknown, nil
	}
}

func inferScanDryingStatus(result ParseLabelResult, evidence careRuleEvidence, normalized string, compact string) (string, *string) {
	temperature := inferDryingTemperature(normalized, compact)
	negativeTumble := containsAny(normalized, "do not tumble dry", "dont tumble dry", "no tumble dry", "no usar secadora") ||
		containsAny(compact, "donottumbledry", "donttumbledry", "notumbledry")
	positiveTumble := containsTumbleDrySignal(normalized, compact)

	switch {
	case containsAny(normalized, "dry flat", "lay flat") || containsAny(compact, "dryflat", "layflat"):
		return dryingStatusDryFlat, temperature
	case containsAny(normalized, "line dry", "hang dry") || containsAny(compact, "linedry", "hangdry"):
		return dryingStatusLineDry, temperature
	case negativeTumble:
		return dryingStatusDoNotTumbleDry, temperature
	case result.TumbleDry || positiveTumble:
		return dryingStatusTumbleDry, temperature
	case evidence.HasDrySignal:
		return dryingStatusDoNotTumbleDry, temperature
	default:
		return dryingStatusUnknown, nil
	}
}

func inferDryingTemperature(normalized string, compact string) *string {
	switch {
	case containsAny(normalized, "tumble dry low", "low heat", "low temperature") || containsAny(compact, "tumbledrylow", "lowheat") ||
		containsTumbleDryTemperature(normalized, "low"):
		return stringPtr("low")
	case containsAny(normalized, "tumble dry medium", "medium heat", "medium temperature") || containsAny(compact, "tumbledrymedium", "mediumheat") ||
		containsTumbleDryTemperature(normalized, "medium"):
		return stringPtr("medium")
	case containsAny(normalized, "tumble dry high", "high heat", "high temperature") || containsAny(compact, "tumbledryhigh", "highheat") ||
		containsTumbleDryTemperature(normalized, "high"):
		return stringPtr("high")
	default:
		return nil
	}
}

func containsTumbleDryTemperature(normalized string, temperature string) bool {
	pattern := regexp.MustCompile(`\btumble(?:\s+\S+){0,5}\s+dry(?:\s+\S+){0,4}\s+` + regexp.QuoteMeta(temperature) + `\b`)
	return pattern.MatchString(normalized)
}

func inferScanIroningStatus(result ParseLabelResult, evidence careRuleEvidence, normalized string, compact string) string {
	negativeIron := containsAny(normalized, "do not iron", "dont iron", "no iron", "no planchar") ||
		containsAny(compact, "donotiron", "dontiron", "noiron", "noplanchar")
	positiveIron := containsIronSignal(normalized, compact)

	switch {
	case negativeIron:
		return ironingStatusDoNotIron
	case result.IronAllowed || positiveIron:
		return ironingStatusAllowed
	case evidence.HasIronSignal:
		return ironingStatusDoNotIron
	default:
		return ironingStatusUnknown
	}
}

func inferScanCleaningStatus(result ParseLabelResult, evidence careRuleEvidence, normalized string, compact string) (string, *string) {
	negativeDryClean := containsAny(normalized, "do not dry clean", "dont dry clean", "no dry clean", "no lavar en seco") ||
		containsAny(compact, "donotdryclean", "dontdryclean", "nodryclean", "nolavarseco")

	switch {
	case negativeDryClean:
		return cleaningStatusDoNotDryClean, nil
	case result.DryCleanOnly:
		return cleaningStatusDryCleanOnly, stringPtr("dry_clean")
	case evidence.HasCleanSignal || containsAny(normalized, "dry clean", "professional clean", "limpieza en seco"):
		return cleaningStatusDryClean, stringPtr("dry_clean")
	default:
		return cleaningStatusUnknown, nil
	}
}

func washSummary(wash WashInstruction) string {
	switch wash.Status {
	case washStatusMachine:
		if wash.MaxTemperatureC != nil {
			return fmt.Sprintf("Machine wash at or below %dC.", *wash.MaxTemperatureC)
		}
		return "Machine wash is allowed."
	case washStatusHand:
		if wash.MaxTemperatureC != nil {
			return fmt.Sprintf("Hand wash at or below %dC.", *wash.MaxTemperatureC)
		}
		return "Hand wash only."
	case washStatusDoNotWash:
		return "Do not wash."
	case washStatusDryCleanOnly:
		return "Do not wash; dry clean only."
	default:
		return "Wash instruction was not detected."
	}
}

func bleachSummary(bleach BleachInstruction) string {
	switch bleach.Status {
	case bleachStatusAllowed:
		return "Bleach is allowed."
	case bleachStatusNonChlorineOnly:
		return "Use non-chlorine bleach only."
	case bleachStatusDoNotBleach:
		return "Do not bleach."
	default:
		return "Bleach instruction was not detected."
	}
}

func dryingSummary(drying DryingInstruction) string {
	switch drying.Status {
	case dryingStatusTumbleDry:
		if drying.Temperature != nil {
			return "Tumble dry on " + *drying.Temperature + " heat."
		}
		return "Tumble dry is allowed."
	case dryingStatusDoNotTumbleDry:
		return "Do not tumble dry."
	case dryingStatusLineDry:
		return "Line dry."
	case dryingStatusDryFlat:
		return "Dry flat."
	default:
		return "Drying instruction was not detected."
	}
}

func ironingSummary(ironing IroningInstruction) string {
	switch ironing.Status {
	case ironingStatusAllowed:
		if ironing.Temperature != nil {
			return "Iron on " + *ironing.Temperature + " heat."
		}
		return "Ironing is allowed."
	case ironingStatusDoNotIron:
		return "Do not iron."
	default:
		return "Ironing instruction was not detected."
	}
}

func cleaningSummary(cleaning ProfessionalCleaningInstruction) string {
	switch cleaning.Status {
	case cleaningStatusDryCleanOnly:
		return "Dry clean only."
	case cleaningStatusDryClean:
		return "Professional dry cleaning is allowed."
	case cleaningStatusDoNotDryClean:
		return "Do not dry clean."
	default:
		return "Professional cleaning instruction was not detected."
	}
}

func defaultScanExplanation(source string, confidence float64) string {
	source = strings.TrimSpace(source)
	if source == "" {
		source = "label parser"
	}
	if confidence < 0.55 {
		return "FreshCycle produced a conservative scan-label result from " + source + "; confirm the label before saving."
	}
	return "FreshCycle inferred structured care instructions from " + source + "."
}

func defaultUncertainFields(result ScanLabelResult) []string {
	fields := make([]string, 0, 8)
	if result.Wash.Status == washStatusUnknown {
		fields = append(fields, "wash")
	}
	if result.Wash.Status == washStatusMachine && result.Wash.MaxTemperatureC == nil {
		fields = append(fields, "wash.max_temperature_c")
	}
	if result.Bleach.Status == bleachStatusUnknown {
		fields = append(fields, "bleach")
	}
	if result.Drying.Status == dryingStatusUnknown {
		fields = append(fields, "drying")
	}
	if result.Ironing.Status == ironingStatusUnknown {
		fields = append(fields, "ironing")
	}
	if result.Ironing.Status == ironingStatusAllowed && result.Ironing.Temperature == nil {
		fields = append(fields, "ironing.temperature")
	}
	if result.ProfessionalCleaning.Status == cleaningStatusUnknown {
		fields = append(fields, "professional_cleaning")
	}
	if strings.TrimSpace(result.RawText) == "" {
		fields = append(fields, "raw_text")
	}
	return fields
}

func defaultConfidenceForScan(result ScanLabelResult) float64 {
	if strings.TrimSpace(result.RawText) == "" {
		return 0.35
	}
	for _, status := range []string{
		result.Wash.Status,
		result.Bleach.Status,
		result.Drying.Status,
		result.Ironing.Status,
		result.ProfessionalCleaning.Status,
	} {
		if status != "" && status != "unknown" {
			return 0.62
		}
	}
	return 0.45
}

func clientEvidenceConfidence(input ScanLabelInput, evidence careRuleEvidence) float64 {
	confidence := normalizeConfidence(0, 0.68)
	if input.ClientOCR != nil && input.ClientOCR.Confidence != nil {
		confidence = normalizeConfidence(*input.ClientOCR.Confidence, 0.68)
	}

	symbolConfidence, ok := averageSymbolConfidence(input.ClientSymbols)
	if detectedConfidence, detectedOK := averageDetectionConfidence(input.DetectedSymbols); detectedOK {
		if ok {
			symbolConfidence = (symbolConfidence + detectedConfidence) / 2
		} else {
			symbolConfidence = detectedConfidence
			ok = true
		}
	}
	if ok {
		if input.ClientOCR != nil && strings.TrimSpace(input.ClientOCR.Text) != "" {
			confidence = confidence*0.7 + symbolConfidence*0.3
		} else {
			confidence = symbolConfidence
		}
	}

	if evidence.SignalCount() < 2 && confidence > 0.58 {
		confidence = 0.58
	}
	return confidence
}

func averageSymbolConfidence(symbols []ClientSymbol) (float64, bool) {
	var total float64
	var count int
	for _, symbol := range symbols {
		if symbol.Confidence == nil {
			continue
		}
		total += normalizeConfidence(*symbol.Confidence, 0.65)
		count++
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

func averageDetectionConfidence(detections []SymbolDetection) (float64, bool) {
	var total float64
	var count int
	for _, detection := range detections {
		total += normalizeConfidence(detection.Confidence, 0.65)
		count++
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

func normalizeConfidence(value float64, fallback float64) float64 {
	if value <= 0 {
		value = fallback
	}
	if value > 1 && value <= 100 {
		value = value / 100
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clientSymbolsToCareText(symbols []ClientSymbol) string {
	phrases := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		class := strings.TrimSpace(symbol.Class)
		if class == "" {
			class, _ = NormalizeLaundrySymbolClass(firstNonEmpty(symbol.Name, symbol.Label))
		}
		if phrase := laundrySymbolImpliedText(class); phrase != "" {
			phrases = append(phrases, phrase)
			if symbolNameImpliesGentleWash(symbol) {
				phrases = append(phrases, "delicate cycle")
			}
			continue
		}

		name := normalizeForRules(strings.NewReplacer("_", " ", "-", " ").Replace(firstNonEmpty(symbol.Name, symbol.Label, symbol.Class)))
		compact := compactForRules(name)
		switch {
		case containsAny(name, "do not wash", "dont wash") || containsAny(compact, "donotwash", "dontwash"):
			phrases = append(phrases, "do not wash")
		case containsAny(name, "hand wash"):
			phrases = append(phrases, "hand wash")
		case containsAny(name, "wash"):
			if temp := inferWashTemperature(name); temp != nil {
				phrases = append(phrases, fmt.Sprintf("wash %d c", *temp))
			} else {
				phrases = append(phrases, "machine wash")
			}
		case containsAny(name, "do not bleach", "dont bleach", "no bleach") || containsAny(compact, "donotbleach", "dontbleach", "nobleach"):
			phrases = append(phrases, "do not bleach")
		case containsAny(name, "non chlorine bleach"):
			phrases = append(phrases, "non chlorine bleach")
		case containsAny(name, "bleach"):
			phrases = append(phrases, "bleach allowed")
		case containsAny(name, "do not tumble dry", "dont tumble dry", "no tumble dry") || containsAny(compact, "donottumbledry", "donttumbledry", "notumbledry"):
			phrases = append(phrases, "do not tumble dry")
		case containsAny(name, "dry flat"):
			phrases = append(phrases, "dry flat")
		case containsAny(name, "line dry"):
			phrases = append(phrases, "line dry")
		case containsAny(name, "tumble dry"):
			phrases = append(phrases, "tumble dry")
		case containsAny(name, "do not iron", "dont iron", "no iron") || containsAny(compact, "donotiron", "dontiron", "noiron"):
			phrases = append(phrases, "do not iron")
		case containsAny(name, "iron low", "cool iron"):
			phrases = append(phrases, "cool iron")
		case containsAny(name, "iron medium", "warm iron"):
			phrases = append(phrases, "warm iron")
		case containsAny(name, "iron high", "hot iron"):
			phrases = append(phrases, "hot iron")
		case containsAny(name, "do not dry clean", "dont dry clean") || containsAny(compact, "donotdryclean", "dontdryclean"):
			phrases = append(phrases, "do not dry clean")
		case containsAny(name, "dry clean only"):
			phrases = append(phrases, "dry clean only")
		case containsAny(name, "dry clean", "professional clean"):
			phrases = append(phrases, "dry clean")
		}
	}
	return strings.Join(phrases, " ")
}

func symbolNameImpliesGentleWash(symbol ClientSymbol) bool {
	name := normalizeForRules(strings.NewReplacer("_", " ", "-", " ").Replace(firstNonEmpty(symbol.Name, symbol.Label, symbol.Class)))
	return containsAny(name, "wash", "tub") && containsAny(name, "underline", "one bar", "gentle", "delicate")
}

func symbolDetectionsToCareText(detections []SymbolDetection) string {
	phrases := make([]string, 0, len(detections))
	for _, detection := range detections {
		if phrase := laundrySymbolImpliedText(detection.Class); phrase != "" {
			phrases = append(phrases, phrase)
		}
	}
	return strings.Join(phrases, " ")
}

func mergeCareEvidence(first careRuleEvidence, second careRuleEvidence) careRuleEvidence {
	return careRuleEvidence{
		HasCareSignal:   first.HasCareSignal || second.HasCareSignal,
		HasWashSignal:   first.HasWashSignal || second.HasWashSignal,
		HasDrySignal:    first.HasDrySignal || second.HasDrySignal,
		HasBleachSignal: first.HasBleachSignal || second.HasBleachSignal,
		HasIronSignal:   first.HasIronSignal || second.HasIronSignal,
		HasCleanSignal:  first.HasCleanSignal || second.HasCleanSignal,
		Conflicting:     first.Conflicting || second.Conflicting,
	}
}

func evidenceFromParseLabelResult(result ParseLabelResult) careRuleEvidence {
	evidence := careRuleEvidence{
		HasWashSignal:   result.MachineWashable || result.WashTempMax != nil || result.DryCleanOnly,
		HasDrySignal:    result.TumbleDry,
		HasBleachSignal: result.BleachAllowed,
		HasIronSignal:   result.IronAllowed || result.IronTemp != nil,
		HasCleanSignal:  result.DryCleanOnly,
	}
	evidence.HasCareSignal = evidence.HasWashSignal || evidence.HasDrySignal || evidence.HasBleachSignal || evidence.HasIronSignal || evidence.HasCleanSignal
	return evidence
}

func validStatus(value string, allowed []string) string {
	value = strings.TrimSpace(value)
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return "unknown"
}

func mergeUniqueStrings(first []string, second []string) []string {
	result := make([]string, 0, len(first)+len(second))
	seen := map[string]bool{}
	for _, values := range [][]string{first, second} {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func appendUniqueString(values []string, next string) []string {
	next = strings.TrimSpace(next)
	if next == "" {
		return values
	}
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstBox(values ...*SymbolBoundingBox) *SymbolBoundingBox {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
