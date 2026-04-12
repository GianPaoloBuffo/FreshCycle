package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/auth"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/garments"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
)

var testAllowedOrigins = []string{"http://localhost:19006", "https://*.vercel.app"}

type stubAuthValidator struct {
	user auth.User
	err  error
}

func (s stubAuthValidator) ValidateAccessToken(_ context.Context, _ string) (auth.User, error) {
	return s.user, s.err
}

type stubGarmentStore struct {
	result garments.Garment
	err    error
	last   garments.CreateInput
}

func (s *stubGarmentStore) CreateGarment(_ context.Context, input garments.CreateInput) (garments.Garment, error) {
	s.last = input
	return s.result, s.err
}

func TestHealthRoute(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, nil).ServeHTTP(recorder, request)

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

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

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

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://app-oqzl4uwro-gpbuffo-5604s-projects.vercel.app" {
		t.Fatalf("expected reflected origin header, got %q", origin)
	}
}

func TestParseLabelRouteRejectsMissingAuth(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/garments/parse-label", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		err: auth.ErrMissingAccessToken,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"auth_required"`)) {
		t.Fatalf("expected auth_required response body, got %s", recorder.Body.String())
	}
}

func TestCreateGarmentRoute(t *testing.T) {
	t.Parallel()

	store := &stubGarmentStore{
		result: garments.Garment{
			ID:               "garment-123",
			UserID:           "user-123",
			Name:             "Navy Hoodie",
			CareInstructions: []string{"Machine washable"},
		},
	}

	requestBody := bytes.NewBufferString(`{
		"name":"Navy Hoodie",
		"category":"Knitwear",
		"primary_color":"Navy",
		"wash_temperature_c":30,
		"care_instructions":["Machine washable","Do not bleach"]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/garments", requestBody)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), store, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	if store.last.UserID != "user-123" {
		t.Fatalf("expected user id user-123, got %s", store.last.UserID)
	}

	var response garments.Garment
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != "garment-123" {
		t.Fatalf("expected garment id garment-123, got %s", response.ID)
	}
}
