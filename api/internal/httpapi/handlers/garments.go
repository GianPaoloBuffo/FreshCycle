package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/garments"
	httpmiddleware "github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi/middleware"
)

type createGarmentRequest struct {
	Name             string   `json:"name"`
	Category         *string  `json:"category"`
	PrimaryColor     *string  `json:"primary_color"`
	WashTemperatureC *int     `json:"wash_temperature_c"`
	CareInstructions []string `json:"care_instructions"`
	LabelImagePath   *string  `json:"label_image_path"`
}

func CreateGarment(store garments.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		user, ok := httpmiddleware.AuthenticatedUserFromContext(request.Context())
		if !ok || strings.TrimSpace(user.ID) == "" {
			writeJSONError(writer, http.StatusUnauthorized, "auth_required", "Sign in again before saving garments.")
			return
		}

		var payload createGarmentRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeJSONError(writer, http.StatusBadRequest, "invalid_json", "FreshCycle could not read the garment payload.")
			return
		}

		garment, err := store.CreateGarment(request.Context(), garments.CreateInput{
			UserID:           user.ID,
			Name:             payload.Name,
			Category:         normalizeOptionalString(payload.Category),
			PrimaryColor:     normalizeOptionalString(payload.PrimaryColor),
			WashTemperatureC: payload.WashTemperatureC,
			CareInstructions: normalizeStringList(payload.CareInstructions),
			LabelImagePath:   normalizeOptionalString(payload.LabelImagePath),
		})
		if err != nil {
			switch {
			case errors.Is(err, garments.ErrNameRequired):
				writeJSONError(writer, http.StatusBadRequest, "name_required", "Add a garment name before saving.")
			case errors.Is(err, garments.ErrWashTemperatureOutOfRange):
				writeJSONError(writer, http.StatusBadRequest, "invalid_wash_temperature", "Wash temperature must be between 0 and 95.")
			default:
				writeJSONError(writer, http.StatusInternalServerError, "garment_save_failed", "FreshCycle could not save that garment just yet.")
			}
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(garment)
	}
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

func writeJSONError(writer http.ResponseWriter, status int, code string, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}
