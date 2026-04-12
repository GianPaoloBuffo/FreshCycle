package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SupabaseValidator struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewSupabaseValidator(baseURL string, apiKey string) SupabaseValidator {
	return SupabaseValidator{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (v SupabaseValidator) ValidateAccessToken(ctx context.Context, accessToken string) (User, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return User{}, ErrMissingAccessToken
	}

	if v.baseURL == "" || v.apiKey == "" {
		return User{}, ErrAuthNotConfigured
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return User{}, fmt.Errorf("build Supabase auth request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("apikey", v.apiKey)

	response, err := v.httpClient.Do(request)
	if err != nil {
		return User{}, fmt.Errorf("call Supabase auth API: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return User{}, ErrInvalidAccessToken
	}

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return User{}, fmt.Errorf("Supabase auth API returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var user User
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&user); err != nil {
		return User{}, fmt.Errorf("decode Supabase auth response: %w", err)
	}

	if strings.TrimSpace(user.ID) == "" {
		return User{}, ErrInvalidAccessToken
	}

	return user, nil
}
