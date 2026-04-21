package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	httpmiddleware "github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi/middleware"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/schedules"
	"github.com/go-chi/chi/v5"
)

type createScheduleRequest struct {
	Name             string   `json:"name"`
	Recurrence       string   `json:"recurrence"`
	StartsOn         *string  `json:"starts_on"`
	GarmentIDs       []string `json:"garment_ids"`
	RemindersEnabled *bool    `json:"reminders_enabled"`
}

func CreateSchedule(store schedules.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		user, ok := httpmiddleware.AuthenticatedUserFromContext(request.Context())
		if !ok || strings.TrimSpace(user.ID) == "" {
			writeJSONError(writer, http.StatusUnauthorized, "auth_required", "Sign in again before saving schedules.")
			return
		}

		var payload createScheduleRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeJSONError(writer, http.StatusBadRequest, "invalid_json", "FreshCycle could not read the schedule payload.")
			return
		}

		remindersEnabled := true
		if payload.RemindersEnabled != nil {
			remindersEnabled = *payload.RemindersEnabled
		}

		schedule, err := store.CreateSchedule(request.Context(), schedules.CreateInput{
			UserID:           user.ID,
			Name:             payload.Name,
			Recurrence:       payload.Recurrence,
			StartsOn:         normalizeOptionalString(payload.StartsOn),
			GarmentIDs:       payload.GarmentIDs,
			RemindersEnabled: remindersEnabled,
		})
		if err != nil {
			switch {
			case errors.Is(err, schedules.ErrNameRequired):
				writeJSONError(writer, http.StatusBadRequest, "schedule_name_required", "Add a schedule name before saving.")
			case errors.Is(err, schedules.ErrGarmentsRequired):
				writeJSONError(writer, http.StatusBadRequest, "garment_ids_required", "Choose at least one garment before saving.")
			case errors.Is(err, schedules.ErrInvalidRecurrence):
				writeJSONError(writer, http.StatusBadRequest, "invalid_recurrence", "Choose a supported recurrence before saving.")
			case errors.Is(err, schedules.ErrInvalidStartDate):
				writeJSONError(writer, http.StatusBadRequest, "invalid_start_date", "Use YYYY-MM-DD for the schedule start date.")
			case errors.Is(err, schedules.ErrInvalidGarmentID):
				writeJSONError(writer, http.StatusBadRequest, "invalid_garment_id", "Choose only garments from your wardrobe before saving.")
			default:
				writeJSONError(writer, http.StatusInternalServerError, "schedule_save_failed", "FreshCycle could not save that schedule just yet.")
			}
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(schedule)
	}
}

func ListSchedules(store schedules.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		user, ok := httpmiddleware.AuthenticatedUserFromContext(request.Context())
		if !ok || strings.TrimSpace(user.ID) == "" {
			writeJSONError(writer, http.StatusUnauthorized, "auth_required", "Sign in again before loading schedules.")
			return
		}

		schedulesList, err := store.ListSchedules(request.Context(), user.ID)
		if err != nil {
			writeJSONError(writer, http.StatusInternalServerError, "schedules_fetch_failed", "FreshCycle could not load your schedules just yet.")
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(schedulesList)
	}
}

func DeleteSchedule(store schedules.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		user, ok := httpmiddleware.AuthenticatedUserFromContext(request.Context())
		if !ok || strings.TrimSpace(user.ID) == "" {
			writeJSONError(writer, http.StatusUnauthorized, "auth_required", "Sign in again before deleting schedules.")
			return
		}

		scheduleID := chi.URLParam(request, "scheduleID")
		if err := store.DeleteSchedule(request.Context(), user.ID, scheduleID); err != nil {
			switch {
			case errors.Is(err, schedules.ErrInvalidID):
				writeJSONError(writer, http.StatusBadRequest, "invalid_schedule_id", "Use a valid schedule id before deleting.")
			case errors.Is(err, schedules.ErrScheduleNotFound):
				writeJSONError(writer, http.StatusNotFound, "schedule_not_found", "FreshCycle could not find that schedule.")
			default:
				writeJSONError(writer, http.StatusInternalServerError, "schedule_delete_failed", "FreshCycle could not delete that schedule just yet.")
			}
			return
		}

		writer.WriteHeader(http.StatusNoContent)
	}
}
