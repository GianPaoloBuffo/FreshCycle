package labelparser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	ocrPSMBlock      = "6"
	ocrPSMSparseText = "11"
)

type OCRExecutor interface {
	RunTSV(ctx context.Context, input ParseLabelInput, pageSegmentationMode string) (string, error)
}

type TesseractRunner struct {
	path      string
	languages string
}

func NewTesseractRunner(path string, languages string) TesseractRunner {
	return TesseractRunner{
		path:      strings.TrimSpace(path),
		languages: strings.TrimSpace(languages),
	}
}

func (r TesseractRunner) RunTSV(ctx context.Context, input ParseLabelInput, pageSegmentationMode string) (string, error) {
	path := r.path
	if path == "" {
		path = "tesseract"
	}

	languages := r.languages
	if languages == "" {
		languages = "eng"
	}

	tempFile, err := os.CreateTemp("", "freshcycle-label-*"+imageExtension(input.MIMEType))
	if err != nil {
		return "", fmt.Errorf("create OCR temp file: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(input.Content); err != nil {
		tempFile.Close()
		return "", fmt.Errorf("write OCR temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("close OCR temp file: %w", err)
	}

	args := []string{
		tempPath,
		"stdout",
		"-l",
		languages,
		"--oem",
		"1",
		"--psm",
		pageSegmentationMode,
		"tsv",
	}
	command := exec.CommandContext(ctx, path, args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr

	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("run tesseract psm %s: %w: %s", pageSegmentationMode, err, truncateForLog(stderr.String(), 400))
	}

	return string(output), nil
}

func imageExtension(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

type OCRParser struct {
	runner OCRExecutor
}

func NewOCRParser(tesseractPath string, languages string) OCRParser {
	return OCRParser{
		runner: NewTesseractRunner(tesseractPath, languages),
	}
}

func NewOCRParserWithRunner(runner OCRExecutor) OCRParser {
	return OCRParser{runner: runner}
}

func (p OCRParser) ParseLabel(ctx context.Context, input ParseLabelInput) (ParseLabelResult, error) {
	details, err := p.ParseLabelWithDetails(ctx, input)
	if err != nil {
		return ParseLabelResult{}, err
	}

	if details.ShouldFallback() && !details.HasUsablePartial() {
		return ParseLabelResult{}, fmt.Errorf("%w: OCR did not produce usable label text", ErrUpstreamParseRejected)
	}

	log.Printf("ocr label parser completed: confidence=%.1f word_count=%d keyword_hits=%d fallback_reasons=%s", details.AverageConfidence, details.WordCount, details.KeywordHits, strings.Join(details.FallbackReasons, ","))
	return details.Result, nil
}

func (p OCRParser) ParseLabelWithDetails(ctx context.Context, input ParseLabelInput) (OCRParseDetails, error) {
	if p.runner == nil {
		return OCRParseDetails{}, ErrProviderUnavailable
	}

	block, blockErr := p.runOCR(ctx, input, ocrPSMBlock)
	sparse, sparseErr := p.runOCR(ctx, input, ocrPSMSparseText)

	if blockErr != nil && sparseErr != nil {
		if errors.Is(blockErr, exec.ErrNotFound) || errors.Is(sparseErr, exec.ErrNotFound) {
			return OCRParseDetails{}, fmt.Errorf("%w: tesseract executable not found", ErrProviderUnavailable)
		}
		return OCRParseDetails{}, fmt.Errorf("%w: OCR failed", ErrUpstreamParseRejected)
	}

	best := chooseOCRRun(block, sparse)
	result, evidence := parseCareLabelText(best.Text)
	details := OCRParseDetails{
		Result:            result,
		Text:              best.Text,
		AverageConfidence: best.AverageConfidence,
		WordCount:         best.WordCount,
		KeywordHits:       best.KeywordHits,
		PSM:               best.PSM,
		Evidence:          evidence,
	}
	details.FallbackReasons = details.evaluateFallbackReasons()

	return details, nil
}

func (p OCRParser) runOCR(ctx context.Context, input ParseLabelInput, psm string) (ocrRun, error) {
	tsv, err := p.runner.RunTSV(ctx, input, psm)
	if err != nil {
		return ocrRun{}, err
	}

	run := parseTesseractTSV(tsv)
	run.PSM = psm
	return run, nil
}

type ocrRun struct {
	Text              string
	AverageConfidence float64
	WordCount         int
	KeywordHits       int
	PSM               string
}

type OCRParseDetails struct {
	Result            ParseLabelResult
	Text              string
	AverageConfidence float64
	WordCount         int
	KeywordHits       int
	PSM               string
	Evidence          careRuleEvidence
	FallbackReasons   []string
}

func (d OCRParseDetails) ShouldFallback() bool {
	return len(d.FallbackReasons) > 0
}

func (d OCRParseDetails) HasUsablePartial() bool {
	return strings.TrimSpace(d.Text) != "" && (d.WordCount >= 3 || d.KeywordHits > 0 || len(d.Result.FabricNotes) > 0)
}

func (d OCRParseDetails) evaluateFallbackReasons() []string {
	reasons := make([]string, 0, 4)
	if strings.TrimSpace(d.Text) == "" || d.WordCount < 3 {
		reasons = append(reasons, "no_useful_text")
	}
	if d.WordCount > 0 && d.AverageConfidence < 55 {
		reasons = append(reasons, "low_ocr_confidence")
	}
	if d.Evidence.Conflicting {
		reasons = append(reasons, "conflicting_care_rules")
	}
	if !d.Evidence.HasCareSignal {
		reasons = append(reasons, "no_care_signal")
	}
	return reasons
}

func chooseOCRRun(first ocrRun, second ocrRun) ocrRun {
	if first.WordCount == 0 {
		return second
	}
	if second.WordCount == 0 {
		return first
	}

	firstScore := ocrRunScore(first)
	secondScore := ocrRunScore(second)
	if secondScore > firstScore {
		return second
	}
	return first
}

func ocrRunScore(run ocrRun) float64 {
	keywordBonus := math.Min(float64(run.KeywordHits)*8, 40)
	wordBonus := math.Min(float64(run.WordCount), 20)
	return run.AverageConfidence + keywordBonus + wordBonus
}

func parseTesseractTSV(tsv string) ocrRun {
	lines := strings.Split(tsv, "\n")
	words := make([]string, 0, len(lines))
	var confidenceTotal float64
	var confidenceCount int

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "level\t") {
			continue
		}

		fields := strings.SplitN(line, "\t", 12)
		if len(fields) < 12 {
			continue
		}

		text := strings.TrimSpace(fields[11])
		if text == "" {
			continue
		}
		words = append(words, text)

		confidence, err := strconv.ParseFloat(strings.TrimSpace(fields[10]), 64)
		if err == nil && confidence >= 0 {
			confidenceTotal += confidence
			confidenceCount++
		}
	}

	text := normalizeOCRText(strings.Join(words, " "))
	averageConfidence := 0.0
	if confidenceCount > 0 {
		averageConfidence = confidenceTotal / float64(confidenceCount)
	}

	return ocrRun{
		Text:              text,
		AverageConfidence: averageConfidence,
		WordCount:         len(words),
		KeywordHits:       countCareKeywords(text),
	}
}

var whitespacePattern = regexp.MustCompile(`\s+`)

func normalizeOCRText(value string) string {
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.ReplaceAll(value, "|", " ")
	value = whitespacePattern.ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func countCareKeywords(text string) int {
	normalized := normalizeForRules(text)
	keywords := []string{
		"wash",
		"machine",
		"hand",
		"bleach",
		"tumble",
		"dry",
		"iron",
		"clean",
		"lavar",
		"lavado",
		"maquina",
		"mano",
		"lejia",
		"blanqueador",
		"secar",
		"secadora",
		"planchar",
		"limpieza",
	}

	hits := 0
	for _, keyword := range keywords {
		if strings.Contains(normalized, keyword) {
			hits++
		}
	}
	return hits
}

func normalizeForRules(value string) string {
	replacer := strings.NewReplacer(
		"á", "a",
		"é", "e",
		"í", "i",
		"ó", "o",
		"ú", "u",
		"ü", "u",
		"ñ", "n",
		"Á", "a",
		"É", "e",
		"Í", "i",
		"Ó", "o",
		"Ú", "u",
		"Ü", "u",
		"Ñ", "n",
	)
	normalized := strings.ToLower(replacer.Replace(value))
	normalized = strings.NewReplacer("°", "", "º", "", "\n", " ", "\t", " ", "/", " ").Replace(normalized)
	return whitespacePattern.ReplaceAllString(normalized, " ")
}
