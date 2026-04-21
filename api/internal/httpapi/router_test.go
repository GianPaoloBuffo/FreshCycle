package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/auth"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/garments"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/schedules"
)

var testAllowedOrigins = []string{"http://localhost:19006", "https://*.vercel.app"}

type stubAuthValidator struct {
	user auth.User
	err  error
}

func (s stubAuthValidator) ValidateAccessToken(_ context.Context, _ string) (auth.User, error) {
	return s.user, s.err
}

type stubGarmentStore struct {
	result          garments.Garment
	getResult       garments.Garment
	list            []garments.Garment
	err             error
	last            garments.CreateInput
	lastUserID      string
	lastGarmentID   string
	lastListOptions garments.ListOptions
}

type stubScheduleStore struct {
	result         schedules.Schedule
	list           []schedules.Schedule
	err            error
	last           schedules.CreateInput
	lastUserID     string
	lastScheduleID string
}

func (s *stubScheduleStore) CreateSchedule(_ context.Context, input schedules.CreateInput) (schedules.Schedule, error) {
	s.last = input
	return s.result, s.err
}

func (s *stubScheduleStore) DeleteSchedule(_ context.Context, userID string, scheduleID string) error {
	s.lastUserID = userID
	s.lastScheduleID = scheduleID
	return s.err
}

func (s *stubScheduleStore) ListSchedules(_ context.Context, userID string) ([]schedules.Schedule, error) {
	s.lastUserID = userID
	return s.list, s.err
}

func (s *stubGarmentStore) CreateGarment(_ context.Context, input garments.CreateInput) (garments.Garment, error) {
	s.last = input
	return s.result, s.err
}

func (s *stubGarmentStore) GetGarment(_ context.Context, userID string, garmentID string) (garments.Garment, error) {
	s.lastUserID = userID
	s.lastGarmentID = garmentID
	return s.getResult, s.err
}

func (s *stubGarmentStore) ListGarments(_ context.Context, userID string, options garments.ListOptions) ([]garments.Garment, error) {
	s.lastUserID = userID
	s.lastListOptions = options
	return s.list, s.err
}

func TestHealthRoute(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, nil).ServeHTTP(recorder, request)

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

func TestParseLabelRoute(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="image"; filename="linen-shirt.jpg"`)
	header.Set("Content-Type", "image/png")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	if _, err := part.Write([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}); err != nil {
		t.Fatalf("write image body: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/garments/parse-label", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"name_suggestion":"Linen Shirt"`)) {
		t.Fatalf("expected stub parse payload, got %s", recorder.Body.String())
	}
}

func TestParseLabelRouteSetsCORSHeadersForVercelPreviewOrigins(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodOptions, "/garments/parse-label", nil)
	request.Header.Set("Origin", "https://app-oqzl4uwro-gpbuffo-5604s-projects.vercel.app")
	request.Header.Set("Access-Control-Request-Method", http.MethodPost)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if origin := recorder.Header().Get("Access-Control-Allow-Origin"); origin != "https://app-oqzl4uwro-gpbuffo-5604s-projects.vercel.app" {
		t.Fatalf("expected reflected origin header, got %q", origin)
	}
}

func TestParseLabelRouteRejectsMissingAuth(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/garments/parse-label", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		err: auth.ErrMissingAccessToken,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"auth_required"`)) {
		t.Fatalf("expected auth_required response body, got %s", recorder.Body.String())
	}
}

func TestCreateGarmentRoute(t *testing.T) {
	t.Parallel()

	store := &stubGarmentStore{
		result: garments.Garment{
			ID:               "garment-123",
			UserID:           "user-123",
			Name:             "Navy Hoodie",
			CareInstructions: []string{"Machine washable"},
		},
	}

	requestBody := bytes.NewBufferString(`{
		"id":"29ce43cd-f095-476d-a7cb-1ee7850c14f1",
		"name":"Navy Hoodie",
		"category":"Knitwear",
		"primary_color":"Navy",
		"wash_temperature_c":30,
		"care_instructions":["Machine washable","Do not bleach"],
		"label_image_path":"user-123/labels/29ce43cd-f095-476d-a7cb-1ee7850c14f1.jpg"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/garments", requestBody)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), store, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	if store.last.UserID != "user-123" {
		t.Fatalf("expected user id user-123, got %s", store.last.UserID)
	}

	if store.last.ID == nil || *store.last.ID != "29ce43cd-f095-476d-a7cb-1ee7850c14f1" {
		t.Fatalf("expected garment id to be forwarded to the store, got %#v", store.last.ID)
	}

	if store.last.LabelImagePath == nil || *store.last.LabelImagePath != "user-123/labels/29ce43cd-f095-476d-a7cb-1ee7850c14f1.jpg" {
		t.Fatalf("expected label image path to be forwarded to the store, got %#v", store.last.LabelImagePath)
	}

	var response garments.Garment
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != "garment-123" {
		t.Fatalf("expected garment id garment-123, got %s", response.ID)
	}
}

func TestCreateGarmentRouteRejectsInvalidLabelImagePath(t *testing.T) {
	t.Parallel()

	requestBody := bytes.NewBufferString(`{
		"id":"29ce43cd-f095-476d-a7cb-1ee7850c14f1",
		"name":"Navy Hoodie",
		"label_image_path":"someone-else/labels/29ce43cd-f095-476d-a7cb-1ee7850c14f1.jpg"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/garments", requestBody)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"invalid_label_image_path"`)) {
		t.Fatalf("expected invalid_label_image_path response body, got %s", recorder.Body.String())
	}
}

func TestListGarmentsRoute(t *testing.T) {
	t.Parallel()

	store := &stubGarmentStore{
		list: []garments.Garment{
			{
				ID:               "garment-123",
				UserID:           "user-123",
				Name:             "Navy Hoodie",
				Category:         pointerTo("Knitwear"),
				PrimaryColor:     pointerTo("Navy"),
				WashTemperatureC: intPointerTo(30),
				CareInstructions: []string{"Machine washable"},
			},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/garments?sort=name&order=asc&category=Knitwear", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), store, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	if store.lastUserID != "user-123" {
		t.Fatalf("expected user id user-123, got %s", store.lastUserID)
	}

	if store.lastListOptions.SortBy != "name" || store.lastListOptions.Order != "asc" {
		t.Fatalf("expected list options to be forwarded, got %#v", store.lastListOptions)
	}

	if store.lastListOptions.Category == nil || *store.lastListOptions.Category != "Knitwear" {
		t.Fatalf("expected category filter Knitwear, got %#v", store.lastListOptions.Category)
	}

	var response []garments.Garment
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 garment, got %d", len(response))
	}

	if response[0].Name != "Navy Hoodie" {
		t.Fatalf("expected garment name Navy Hoodie, got %s", response[0].Name)
	}
}

func TestListGarmentsRouteRejectsMissingAuth(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/garments", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		err: auth.ErrMissingAccessToken,
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"auth_required"`)) {
		t.Fatalf("expected auth_required response body, got %s", recorder.Body.String())
	}
}

func TestListGarmentsRouteRejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/garments?sort=temperature", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"invalid_query"`)) {
		t.Fatalf("expected invalid_query response body, got %s", recorder.Body.String())
	}
}

func TestGetGarmentRoute(t *testing.T) {
	t.Parallel()

	store := &stubGarmentStore{
		getResult: garments.Garment{
			ID:               "29ce43cd-f095-476d-a7cb-1ee7850c14f1",
			UserID:           "user-123",
			Name:             "Navy Hoodie",
			CareInstructions: []string{"Machine washable"},
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/garments/29ce43cd-f095-476d-a7cb-1ee7850c14f1", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), store, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	if store.lastUserID != "user-123" || store.lastGarmentID != "29ce43cd-f095-476d-a7cb-1ee7850c14f1" {
		t.Fatalf("expected ownership-safe garment lookup, got user=%q garment=%q", store.lastUserID, store.lastGarmentID)
	}
}

func TestGetGarmentRouteRejectsInvalidID(t *testing.T) {
	t.Parallel()

	store := &stubGarmentStore{
		err: garments.ErrInvalidID,
	}

	request := httptest.NewRequest(http.MethodGet, "/garments/not-a-uuid", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), store, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"invalid_garment_id"`)) {
		t.Fatalf("expected invalid_garment_id response body, got %s", recorder.Body.String())
	}
}

func TestGetGarmentRouteReturnsNotFound(t *testing.T) {
	t.Parallel()

	store := &stubGarmentStore{
		err: garments.ErrGarmentNotFound,
	}

	request := httptest.NewRequest(http.MethodGet, "/garments/29ce43cd-f095-476d-a7cb-1ee7850c14f1", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), store, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"garment_not_found"`)) {
		t.Fatalf("expected garment_not_found response body, got %s", recorder.Body.String())
	}
}

func TestCreateScheduleRoute(t *testing.T) {
	t.Parallel()

	store := &stubScheduleStore{
		result: schedules.Schedule{
			ID:               "schedule-123",
			UserID:           "user-123",
			Name:             "Weekly towels",
			Recurrence:       "weekly:monday",
			GarmentIDs:       []string{"29ce43cd-f095-476d-a7cb-1ee7850c14f1"},
			RemindersEnabled: true,
			CreatedAt:        time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC),
		},
	}

	requestBody := bytes.NewBufferString(`{
		"name":"Weekly towels",
		"recurrence":"weekly:monday",
		"garment_ids":["29ce43cd-f095-476d-a7cb-1ee7850c14f1"],
		"reminders_enabled":true
	}`)
	request := httptest.NewRequest(http.MethodPost, "/schedules", requestBody)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	if store.last.UserID != "user-123" {
		t.Fatalf("expected user id user-123, got %s", store.last.UserID)
	}

	if store.last.Name != "Weekly towels" || store.last.Recurrence != "weekly:monday" {
		t.Fatalf("expected schedule fields to be forwarded, got %#v", store.last)
	}

	if len(store.last.GarmentIDs) != 1 || store.last.GarmentIDs[0] != "29ce43cd-f095-476d-a7cb-1ee7850c14f1" {
		t.Fatalf("expected garment ids to be forwarded, got %#v", store.last.GarmentIDs)
	}

	var response schedules.Schedule
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ID != "schedule-123" {
		t.Fatalf("expected schedule id schedule-123, got %s", response.ID)
	}
}

func TestCreateScheduleRouteRejectsInvalidRecurrence(t *testing.T) {
	t.Parallel()

	store := &stubScheduleStore{
		err: schedules.ErrInvalidRecurrence,
	}
	requestBody := bytes.NewBufferString(`{
		"name":"Weekly towels",
		"recurrence":"monthly",
		"garment_ids":["29ce43cd-f095-476d-a7cb-1ee7850c14f1"]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/schedules", requestBody)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"invalid_recurrence"`)) {
		t.Fatalf("expected invalid_recurrence response body, got %s", recorder.Body.String())
	}
}

func TestListSchedulesRoute(t *testing.T) {
	t.Parallel()

	store := &stubScheduleStore{
		list: []schedules.Schedule{
			{
				ID:               "schedule-123",
				UserID:           "user-123",
				Name:             "Weekly towels",
				Recurrence:       "weekly:monday",
				GarmentIDs:       []string{"29ce43cd-f095-476d-a7cb-1ee7850c14f1"},
				RemindersEnabled: true,
				CreatedAt:        time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/schedules", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	if store.lastUserID != "user-123" {
		t.Fatalf("expected user id user-123, got %s", store.lastUserID)
	}

	var response []schedules.Schedule
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response) != 1 || response[0].Name != "Weekly towels" {
		t.Fatalf("expected saved schedule response, got %#v", response)
	}
}

func TestListSchedulesRouteRejectsMissingAuth(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/schedules", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		err: auth.ErrMissingAccessToken,
	}, &stubScheduleStore{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"auth_required"`)) {
		t.Fatalf("expected auth_required response body, got %s", recorder.Body.String())
	}
}

func TestDeleteScheduleRoute(t *testing.T) {
	t.Parallel()

	store := &stubScheduleStore{}
	request := httptest.NewRequest(http.MethodDelete, "/schedules/29ce43cd-f095-476d-a7cb-1ee7850c14f1", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}

	if store.lastUserID != "user-123" || store.lastScheduleID != "29ce43cd-f095-476d-a7cb-1ee7850c14f1" {
		t.Fatalf("expected ownership-safe delete, got user=%q schedule=%q", store.lastUserID, store.lastScheduleID)
	}
}

func TestDeleteScheduleRouteReturnsNotFound(t *testing.T) {
	t.Parallel()

	store := &stubScheduleStore{
		err: schedules.ErrScheduleNotFound,
	}
	request := httptest.NewRequest(http.MethodDelete, "/schedules/29ce43cd-f095-476d-a7cb-1ee7850c14f1", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"schedule_not_found"`)) {
		t.Fatalf("expected schedule_not_found response body, got %s", recorder.Body.String())
	}
}

func pointerTo(value string) *string {
	return &value
}

func intPointerTo(value int) *int {
	return &value
}
