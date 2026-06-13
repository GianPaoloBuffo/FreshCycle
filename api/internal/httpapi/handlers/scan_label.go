package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
)

var (
	errInvalidClientOCR     = errors.New("invalid client OCR JSON")
	errInvalidClientSymbols = errors.New("invalid client symbols JSON")
)

func ScanLabel(parser labelparser.Parser) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		input, err := readScanLabelInput(request)
		if err != nil {
			writeScanLabelError(writer, err)
			return
		}

		result, err := labelparser.ScanLabel(request.Context(), parser, input)
		if err != nil {
			writeScanLabelError(writer, err)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(result)
	}
}

func readScanLabelInput(request *http.Request) (labelparser.ScanLabelInput, error) {
	image, err := readLabelImage(request)
	if err != nil {
		return labelparser.ScanLabelInput{}, err
	}

	clientOCR, err := readClientOCR(request)
	if err != nil {
		return labelparser.ScanLabelInput{}, err
	}

	clientSymbols, err := readClientSymbols(request)
	if err != nil {
		return labelparser.ScanLabelInput{}, err
	}

	return labelparser.ScanLabelInput{
		ParseLabelInput: image,
		ClientOCR:       clientOCR,
		ClientSymbols:   clientSymbols,
	}, nil
}

func readClientOCR(request *http.Request) (*labelparser.ClientOCR, error) {
	raw := optionalMultipartValue(request, "client_ocr")
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var clientOCR labelparser.ClientOCR
	if err := json.Unmarshal([]byte(raw), &clientOCR); err != nil {
		return nil, errInvalidClientOCR
	}
	return &clientOCR, nil
}

func readClientSymbols(request *http.Request) ([]labelparser.ClientSymbol, error) {
	raw := optionalMultipartValue(request, "client_symbols")
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var symbols []labelparser.ClientSymbol
	if err := json.Unmarshal([]byte(raw), &symbols); err != nil {
		var wrapper struct {
			Symbols []labelparser.ClientSymbol `json:"symbols"`
		}
		if wrapperErr := json.Unmarshal([]byte(raw), &wrapper); wrapperErr != nil {
			return nil, errInvalidClientSymbols
		}
		symbols = wrapper.Symbols
	}

	normalized := make([]labelparser.ClientSymbol, 0, len(symbols))
	for _, symbol := range symbols {
		if strings.TrimSpace(symbol.Name) == "" {
			continue
		}
		normalized = append(normalized, symbol)
	}
	return normalized, nil
}

func optionalMultipartValue(request *http.Request, field string) string {
	if request.MultipartForm == nil {
		return strings.TrimSpace(request.FormValue(field))
	}

	values := request.MultipartForm.Value[field]
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func writeScanLabelError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidClientOCR):
		writeJSONError(writer, http.StatusBadRequest, "invalid_client_ocr", "client_ocr must be valid JSON with readable label text.")
	case errors.Is(err, errInvalidClientSymbols):
		writeJSONError(writer, http.StatusBadRequest, "invalid_client_symbols", "client_symbols must be valid JSON.")
	default:
		writeParseLabelError(writer, err)
	}
}
