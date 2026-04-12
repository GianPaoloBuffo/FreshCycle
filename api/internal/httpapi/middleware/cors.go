package middleware

import (
	"net/http"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins []string
}

func CORS(config CORSConfig) func(http.Handler) http.Handler {
	allowedOrigins := append([]string(nil), config.AllowedOrigins...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			origin := strings.TrimSpace(request.Header.Get("Origin"))
			if matchedOrigin, ok := matchAllowedOrigin(origin, allowedOrigins); ok {
				writer.Header().Set("Access-Control-Allow-Origin", matchedOrigin)
				writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				writer.Header().Set("Access-Control-Allow-Credentials", "true")
				writer.Header().Set("Vary", "Origin")
			}

			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

func matchAllowedOrigin(origin string, allowedOrigins []string) (string, bool) {
	if origin == "" {
		return "", false
	}

	for _, allowedOrigin := range allowedOrigins {
		allowedOrigin = strings.TrimSpace(allowedOrigin)
		if allowedOrigin == "" {
			continue
		}

		if allowedOrigin == "*" || origin == allowedOrigin {
			return origin, true
		}

		if strings.Contains(allowedOrigin, "*.") {
			parts := strings.SplitN(allowedOrigin, "*", 2)
			prefix := parts[0]
			suffix := ""
			if len(parts) == 2 {
				suffix = parts[1]
			}
			if strings.HasPrefix(origin, prefix) && strings.HasSuffix(origin, suffix) {
				return origin, true
			}
		}
	}

	return "", false
}
