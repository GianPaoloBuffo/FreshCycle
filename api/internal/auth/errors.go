package auth

import "errors"

var (
	ErrMissingAccessToken = errors.New("missing access token")
	ErrInvalidAccessToken = errors.New("invalid access token")
	ErrAuthNotConfigured  = errors.New("auth validator not configured")
)
