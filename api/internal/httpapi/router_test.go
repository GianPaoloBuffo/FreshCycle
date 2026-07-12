package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/GianPaoloBuffo/FreshCycle/api/internal/auth"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/garments"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/httpapi"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/labelparser"
	"github.com/GianPaoloBuffo/FreshCycle/api/internal/scanquality"
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

type stubScanQualityStore struct {
	scanResult     scanquality.ScanRecordResult
	decisionResult scanquality.ReviewDecisionResult
	lastScan       scanquality.ScanRecordInput
	lastDecision   scanquality.ReviewDecisionInput
	err            error
}

func (s *stubScanQualityStore) RecordScan(_ context.Context, input scanquality.ScanRecordInput) (scanquality.ScanRecordResult, error) {
	s.lastScan = input
	return s.scanResult, s.err
}

func (s *stubScanQualityStore) RecordReviewDecision(_ context.Context, input scanquality.ReviewDecisionInput) (scanquality.ReviewDecisionResult, error) {
	s.lastDecision = input
	return s.decisionResult, s.err
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

	request := newMultipartImageRequest(t, "/garments/parse-label", nil)
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

func TestScanLabelRoute(t *testing.T) {
	t.Parallel()

	request := newMultipartImageRequest(t, "/scan-label", nil)
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response labelparser.ScanLabelResult
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode scan-label response: %v", err)
	}

	if response.Wash.Status != "machine_wash" {
		t.Fatalf("expected machine wash status, got %#v", response.Wash)
	}
	if response.Bleach.Status != "do_not_bleach" {
		t.Fatalf("expected do not bleach status, got %#v", response.Bleach)
	}
	if response.RawText == "" || response.Confidence <= 0 || response.Explanation == "" {
		t.Fatalf("expected stable scan metadata, got %#v", response)
	}
	if response.UncertainFields == nil {
		t.Fatal("expected uncertain_fields to be present")
	}
}

func TestScanLabelRouteRecordsTelemetry(t *testing.T) {
	t.Parallel()

	store := &stubScanQualityStore{
		scanResult: scanquality.ScanRecordResult{
			ScanEventID:            "29ce43cd-f095-476d-a7cb-1ee7850c14f1",
			ReviewQueueID:          "50f9b725-df5a-4508-a2fd-34ceb4675ae4",
			ReviewReasons:          []string{"low_confidence"},
			ActiveLearningPriority: 0.74,
		},
	}
	request := newMultipartImageRequest(t, "/scan-label", map[string]string{
		"capture_source":  "camera",
		"image_scope":     "label_crop",
		"capture_quality": `{"score":0.62,"hints":[{"issue":"low_ocr_signal"}]}`,
	})
	recorder := httptest.NewRecorder()

	httpapi.NewRouterWithScanQuality(labelparser.NewStubParser(), &stubGarmentStore{}, store, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	if store.lastScan.UserID != "user-123" || store.lastScan.Outcome != scanquality.OutcomeSuccess {
		t.Fatalf("expected scan telemetry to include authenticated user and success outcome, got %#v", store.lastScan)
	}
	if store.lastScan.Request.CaptureSource != "camera" {
		t.Fatalf("expected capture source to be forwarded, got %#v", store.lastScan.Request)
	}
	if store.lastScan.Request.CaptureQualityScore == nil || *store.lastScan.Request.CaptureQualityScore != 0.62 {
		t.Fatalf("expected capture quality score to be forwarded, got %#v", store.lastScan.Request.CaptureQualityScore)
	}

	var response labelparser.ScanLabelResult
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode scan-label response: %v", err)
	}

	if response.ScanEventID != store.scanResult.ScanEventID || response.ReviewQueueID != store.scanResult.ReviewQueueID {
		t.Fatalf("expected telemetry identifiers in response, got %#v", response)
	}
	if len(response.ReviewReasons) != 1 || response.ReviewReasons[0] != "low_confidence" {
		t.Fatalf("expected review reasons in response, got %#v", response.ReviewReasons)
	}
}

func TestScanReviewEventRouteRecordsDecision(t *testing.T) {
	t.Parallel()

	store := &stubScanQualityStore{
		decisionResult: scanquality.ReviewDecisionResult{
			ReviewDecisionID:    "b8261c04-f11a-42c8-9ddb-7afd031c5ba2",
			ReviewQueueID:       "50f9b725-df5a-4508-a2fd-34ceb4675ae4",
			AnnotationExampleID: "dd69e3da-b43d-4272-9d95-36d704709029",
		},
	}
	requestBody := bytes.NewBufferString(`{
		"scan_event_id":"29ce43cd-f095-476d-a7cb-1ee7850c14f1",
		"review_queue_id":"50f9b725-df5a-4508-a2fd-34ceb4675ae4",
		"decision":"correct",
		"corrected_fields":["bleach"],
		"field_corrections":[{"field_name":"bleach","predicted_value":"allowed","corrected_value":"do_not_bleach","error_source":"user_override","confidence":0.71}],
		"final_user_correction":{"bleach":"do_not_bleach"}
	}`)
	request := httptest.NewRequest(http.MethodPost, "/scan-label/review-events", requestBody)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	httpapi.NewRouterWithScanQuality(labelparser.NewStubParser(), &stubGarmentStore{}, store, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if store.lastDecision.UserID != "user-123" || store.lastDecision.Decision != scanquality.DecisionCorrect {
		t.Fatalf("expected review decision to be forwarded, got %#v", store.lastDecision)
	}
	if len(store.lastDecision.FieldCorrections) != 1 || store.lastDecision.FieldCorrections[0].FieldName != "bleach" {
		t.Fatalf("expected field correction payload, got %#v", store.lastDecision.FieldCorrections)
	}
}

func TestScanLabelRouteUsesClientOCR(t *testing.T) {
	t.Parallel()

	request := newMultipartImageRequest(t, "/scan-label", map[string]string{
		"client_ocr": `{"text":"Dry clean only. Do not wash. Do not tumble dry. Do not bleach.","confidence":0.92}`,
	})
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response labelparser.ScanLabelResult
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode scan-label response: %v", err)
	}

	if response.Wash.Status != "dry_clean_only" {
		t.Fatalf("expected client OCR to drive wash status, got %#v", response.Wash)
	}
	if response.ProfessionalCleaning.Status != "dry_clean_only" {
		t.Fatalf("expected dry clean only status, got %#v", response.ProfessionalCleaning)
	}
	if response.Confidence < 0.9 {
		t.Fatalf("expected client OCR confidence to be preserved, got %f", response.Confidence)
	}
}

func TestScanLabelRouteRejectsMissingAuth(t *testing.T) {
	t.Parallel()

	request := newMultipartImageRequest(t, "/scan-label", nil)
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

func TestScanLabelRouteRejectsMissingImage(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("client_ocr", `{"text":"Machine wash cold"}`); err != nil {
		t.Fatalf("write field: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/scan-label", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()

	httpapi.NewRouter(labelparser.NewStubParser(), &stubGarmentStore{}, testAllowedOrigins, stubAuthValidator{
		user: auth.User{ID: "user-123"},
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}

	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"error":"image_required"`)) {
		t.Fatalf("expected image_required response body, got %s", recorder.Body.String())
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

	if methods := recorder.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, http.MethodDelete) {
		t.Fatalf("expected DELETE in allowed methods, got %q", methods)
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
			StartsOn:         pointerTo("2026-04-20"),
			GarmentIDs:       []string{"29ce43cd-f095-476d-a7cb-1ee7850c14f1"},
			RemindersEnabled: true,
			CreatedAt:        time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC),
		},
	}

	requestBody := bytes.NewBufferString(`{
		"name":"Weekly towels",
		"recurrence":"weekly:monday",
		"starts_on":"2026-04-20",
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

	if store.last.StartsOn == nil || *store.last.StartsOn != "2026-04-20" {
		t.Fatalf("expected starts_on to be forwarded, got %#v", store.last.StartsOn)
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
				StartsOn:         pointerTo("2026-04-20"),
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

func newMultipartImageRequest(t *testing.T, path string, fields map[string]string) *http.Request {
	t.Helper()

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

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write multipart field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func pointerTo(value string) *string {
	return &value
}

func intPointerTo(value int) *int {
	return &value
}
