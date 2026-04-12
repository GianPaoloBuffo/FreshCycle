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
	ID               *string  `json:"id"`
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

		if path := normalizeOptionalString(payload.LabelImagePath); path != nil {
			if !strings.HasPrefix(*path, user.ID+"/labels/") {
				writeJSONError(writer, http.StatusBadRequest, "invalid_label_image_path", "Label uploads must stay inside your private labels folder.")
				return
			}
		}

		garment, err := store.CreateGarment(request.Context(), garments.CreateInput{
			ID:               normalizeOptionalString(payload.ID),
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
			case errors.Is(err, garments.ErrInvalidID):
				writeJSONError(writer, http.StatusBadRequest, "invalid_garment_id", "Use a valid garment id before saving.")
			case errors.Is(err, garments.ErrNameRequired):
				writeJSONError(writer, http.StatusBadRequest, "name_required", "Add a garment name before saving.")
			case errors.Is(err, garments.ErrWashTemperatureOutOfRange):
				writeJSONError(writer, http.StatusBadRequest, "invalid_wash_temperature", "Wash temperature must be between 0 and 95.")
			case errors.Is(err, garments.ErrInvalidLabelImagePath):
				writeJSONError(writer, http.StatusBadRequest, "invalid_label_image_path", "Label uploads must stay inside your private labels folder.")
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

func ListGarments(store garments.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		user, ok := httpmiddleware.AuthenticatedUserFromContext(request.Context())
		if !ok || strings.TrimSpace(user.ID) == "" {
			writeJSONError(writer, http.StatusUnauthorized, "auth_required", "Sign in again before loading garments.")
			return
		}

		garmentsList, err := store.ListGarments(request.Context(), user.ID)
		if err != nil {
			writeJSONError(writer, http.StatusInternalServerError, "garments_fetch_failed", "FreshCycle could not load your wardrobe just yet.")
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(garmentsList)
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
