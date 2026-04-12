package httpapi_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
)

var testAllowedOrigins = []string{"http://localhost:19006", "https://*.vercel.app"}

func TestHealthRoute(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), testAllowedOrigins).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	expectedBody := "{\"status\":\"ok\"}\n"
	if recorder.Body.String() != expectedBody {
		t.Fatalf("expected body %q, got %q", expectedBody, recorder.Body.String())
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("expected content type application/json, got %q", contentType)
	}
}

func TestParseLabelRoute(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="image"; filename="linen-shirt.jpg"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, err := part.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}); err != nil {
		t.Fatalf("write image body: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/garments/parse-label", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), testAllowedOrigins).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"name_suggestion":"Linen Shirt"`)) {
		t.Fatalf("expected stub parse payload, got %s", recorder.Body.String())
	}
}

func TestParseLabelRouteSetsCORSHeadersForVercelPreviewOrigins(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodOptions, "/garments/parse-label", nil)
	request.Header.Set("Origin", "https://app-oqzl4uwro-gpbuffo-5604s-projects.vercel.app")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), testAllowedOrigins).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://app-oqzl4uwro-gpbuffo-5604s-projects.vercel.app" {
		t.Fatalf("expected reflected origin header, got %q", origin)
	}
}
