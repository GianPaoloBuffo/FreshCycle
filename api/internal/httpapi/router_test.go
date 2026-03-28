package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi"
)

func TestHealthRoute(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter().ServeHTTP(recorder, request)

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
