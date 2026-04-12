package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSupabaseValidatorValidatesUserFromAuthAPI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("expected Authorization header, got %q", got)
		}

		if got := request.Header.Get("apikey"); got != "secret-key" {
			t.Fatalf("expected apikey header, got %q", got)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"user-123","email":"test@example.com","role":"authenticated"}`))
	}))
	defer server.Close()

	validator := NewSupabaseValidator(server.URL, "secret-key")

	user, err := validator.ValidateAccessToken(t.Context(), "token-123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if user.ID != "user-123" {
		t.Fatalf("expected user id user-123, got %s", user.ID)
	}
}

func TestSupabaseValidatorRejectsUnauthorizedToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	validator := NewSupabaseValidator(server.URL, "secret-key")

	if _, err := validator.ValidateAccessToken(t.Context(), "bad-token"); err != ErrInvalidAccessToken {
		t.Fatalf("expected ErrInvalidAccessToken, got %v", err)
	}
}
