package labelparser

import (
	"context"
	"encoding/base64"
	"errors"
	"os/exec"
	"testing"
)

func TestParseTesseractTSVNormalizesTextAndConfidence(t *testing.T) {
	run := parseTesseractTSV(sampleTSV(
		tsvWord{"Machine", "80"},
		tsvWord{"wash", "70"},
		tsvWord{"cold", "60"},
	))

	if run.Text != "Machine wash cold" {
		t.Fatalf("expected normalized text, got %q", run.Text)
	}
	if run.WordCount != 3 {
		t.Fatalf("expected 3 words, got %d", run.WordCount)
	}
	if run.AverageConfidence != 70 {
		t.Fatalf("expected average confidence 70, got %f", run.AverageConfidence)
	}
	if run.KeywordHits == 0 {
		t.Fatal("expected care keyword hits")
	}
}

func TestChooseOCRRunBalancesConfidenceAndCareKeywords(t *testing.T) {
	block := ocrRun{
		Text:              "random product copy",
		AverageConfidence: 92,
		WordCount:         3,
		KeywordHits:       0,
		PSM:               ocrPSMBlock,
	}
	sparse := ocrRun{
		Text:              "machine wash cold do not bleach",
		AverageConfidence: 70,
		WordCount:         6,
		KeywordHits:       4,
		PSM:               ocrPSMSparseText,
	}

	if got := chooseOCRRun(block, sparse); got.PSM != ocrPSMSparseText {
		t.Fatalf("expected sparse OCR run to win, got psm %q", got.PSM)
	}
}

func TestOCRParserUsesBestTSVAndParsesCareRules(t *testing.T) {
	parser := NewOCRParserWithRunner(fakeOCRRunner{
		responses: map[string]string{
			ocrPSMBlock:      sampleTSV(tsvWord{"style", "90"}, tsvWord{"abc", "90"}),
			ocrPSMSparseText: sampleTSV(tsvWord{"Machine", "80"}, tsvWord{"wash", "80"}, tsvWord{"cold", "80"}, tsvWord{"Do", "80"}, tsvWord{"not", "80"}, tsvWord{"bleach", "80"}),
		},
	})

	details, err := parser.ParseLabelWithDetails(context.Background(), ParseLabelInput{
		Filename: "label.jpg",
		MIMEType: "image/jpeg",
		Content:  []byte("image"),
	})
	if err != nil {
		t.Fatalf("parse label with OCR: %v", err)
	}

	if details.PSM != ocrPSMSparseText {
		t.Fatalf("expected sparse psm, got %q", details.PSM)
	}
	if details.Result.WashTempMax == nil || *details.Result.WashTempMax != 30 {
		t.Fatalf("expected cold wash to infer 30C, got %#v", details.Result.WashTempMax)
	}
	if !details.Result.MachineWashable {
		t.Fatal("expected machine washable")
	}
	if details.Result.BleachAllowed {
		t.Fatal("expected bleach disallowed")
	}
	if details.ShouldFallback() {
		t.Fatalf("did not expect fallback reasons, got %#v", details.FallbackReasons)
	}
}

func TestOCRParseDetailsFallbackReasons(t *testing.T) {
	result, evidence := parseCareLabelText("RN 12345 logo")
	details := OCRParseDetails{
		Result:            result,
		Text:              "RN 12345 logo",
		AverageConfidence: 35,
		WordCount:         3,
		KeywordHits:       0,
		Evidence:          evidence,
	}

	reasons := details.evaluateFallbackReasons()
	if len(reasons) != 2 || reasons[0] != "low_ocr_confidence" || reasons[1] != "no_care_signal" {
		t.Fatalf("unexpected fallback reasons: %#v", reasons)
	}
}

func TestOCRParseDetailsFlagsPartialCareLabel(t *testing.T) {
	result, evidence := parseCareLabelText("MACHINE WASH eon")
	details := OCRParseDetails{
		Result:            result,
		Text:              "MACHINE WASH eon",
		AverageConfidence: 82,
		WordCount:         3,
		KeywordHits:       2,
		Evidence:          evidence,
	}

	reasons := details.evaluateFallbackReasons()
	if len(reasons) != 1 || reasons[0] != "partial_care_label" {
		t.Fatalf("expected partial_care_label fallback reason, got %#v", reasons)
	}
}

func TestTesseractRunnerIntegrationSkipsWhenUnavailable(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract is not installed")
	}

	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png fixture: %v", err)
	}

	runner := NewTesseractRunner("tesseract", "eng")
	_, err = runner.RunTSV(context.Background(), ParseLabelInput{
		Filename: "blank.png",
		MIMEType: "image/png",
		Content:  pngBytes,
	}, ocrPSMBlock)
	if err != nil {
		t.Fatalf("run tesseract: %v", err)
	}
}

type fakeOCRRunner struct {
	responses map[string]string
	err       error
}

func (r fakeOCRRunner) RunTSV(_ context.Context, _ ParseLabelInput, pageSegmentationMode string) (string, error) {
	if r.err != nil {
		return "", r.err
	}

	response, ok := r.responses[pageSegmentationMode]
	if !ok {
		return "", errors.New("missing fake OCR response")
	}
	return response, nil
}

type tsvWord struct {
	text string
	conf string
}

func sampleTSV(words ...tsvWord) string {
	result := "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n"
	for index, word := range words {
		result += "5\t1\t1\t1\t1\t" + string(rune('1'+index)) + "\t0\t0\t10\t10\t" + word.conf + "\t" + word.text + "\n"
	}
	return result
}
