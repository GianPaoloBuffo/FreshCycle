package auth

import "github.com/GianPaoloBuffo/FreshCycle/api/internal/config"

func NewValidator(cfg config.Config) Validator {
	if cfg.SupabaseProjectURL == "" || cfg.SupabaseSecretKey == "" {
		return nil
	}

	validator := NewSupabaseValidator(cfg.SupabaseProjectURL, cfg.SupabaseSecretKey)
	return validator
}
