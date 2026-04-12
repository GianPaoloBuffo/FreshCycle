package auth

import "context"

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Validator interface {
	ValidateAccessToken(ctx context.Context, accessToken string) (User, error)
}
