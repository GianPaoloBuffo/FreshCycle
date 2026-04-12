package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
)

const maxLabelUploadBytes = 10 << 20

func ParseLabel(parser labelparser.Parser) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		image, err := readLabelImage(request)
		if err != nil {
			writeParseLabelError(writer, err)
			return
		}

		result, err := parser.ParseLabel(request.Context(), image)
		if err != nil {
			writeParseLabelError(writer, err)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(result)
	}
}

func readLabelImage(request *http.Request) (labelparser.ParseLabelInput, error) {
	if err := request.ParseMultipartForm(maxLabelUploadBytes); err != nil {
		return labelparser.ParseLabelInput{}, labelparser.ErrImageRequired
	}

	file, header, err := request.FormFile("image")
	if err != nil {
		return labelparser.ParseLabelInput{}, labelparser.ErrImageRequired
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxLabelUploadBytes))
	if err != nil {
		return labelparser.ParseLabelInput{}, err
	}

	mimeType := detectImageMIMEType(content, header.Header.Get("Content-Type"))
	if mimeType == "" {
		return labelparser.ParseLabelInput{}, labelparser.ErrUnsupportedImageType
	}

	return labelparser.ParseLabelInput{
		Filename: header.Filename,
		MIMEType: mimeType,
		Content:  content,
	}, nil
}

func detectImageMIMEType(content []byte, declared string) string {
	sniffed := http.DetectContentType(content)
	switch {
	case strings.HasPrefix(sniffed, "image/"):
		return sniffed
	case strings.HasPrefix(strings.TrimSpace(declared), "image/"):
		return strings.TrimSpace(declared)
	default:
		return ""
	}
}

func writeParseLabelError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "parse_failed"
	message := "FreshCycle could not parse that care-label image."

	switch {
	case errors.Is(err, labelparser.ErrImageRequired):
		status = http.StatusBadRequest
		code = "image_required"
		message = "Attach a care-label image in the multipart form field named image."
	case errors.Is(err, labelparser.ErrUnsupportedImageType):
		status = http.StatusUnsupportedMediaType
		code = "unsupported_image_type"
		message = "Only image uploads are supported for label parsing."
	case errors.Is(err, labelparser.ErrProviderUnavailable):
		status = http.StatusServiceUnavailable
		code = "provider_unavailable"
		message = "The configured label parser provider is not ready yet."
	case errors.Is(err, labelparser.ErrUpstreamParseRejected):
		status = http.StatusBadGateway
		code = "provider_error"
		message = "The label parser provider could not parse that image."
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}
