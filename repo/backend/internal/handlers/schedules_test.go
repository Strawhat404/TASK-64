package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Schedule handler tests ---
// Tests request validation and parameter parsing for scheduling endpoints.

func TestCreateScheduleInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/schedules", strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Scheduler")

	h := &ScheduleHandler{DB: nil}
	err := h.CreateSchedule(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateScheduleMissingFields(t *testing.T) {
	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			"missing service_id",
			map[string]interface{}{
				"staff_id":        uuid.New().String(),
				"client_name":     "John Doe",
				"scheduled_start": time.Now().Add(time.Hour).Format(time.RFC3339),
				"scheduled_end":   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			"missing staff_id",
			map[string]interface{}{
				"service_id":      uuid.New().String(),
				"client_name":     "John Doe",
				"scheduled_start": time.Now().Add(time.Hour).Format(time.RFC3339),
				"scheduled_end":   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			"missing client_name",
			map[string]interface{}{
				"service_id":      uuid.New().String(),
				"staff_id":        uuid.New().String(),
				"scheduled_start": time.Now().Add(time.Hour).Format(time.RFC3339),
				"scheduled_end":   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/schedules", strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Scheduler")

			h := &ScheduleHandler{DB: nil}
			_ = h.CreateSchedule(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestCreateScheduleEndBeforeStart(t *testing.T) {
	e := echo.New()
	now := time.Now().Add(2 * time.Hour)
	b, _ := json.Marshal(map[string]interface{}{
		"service_id":      uuid.New().String(),
		"staff_id":        uuid.New().String(),
		"client_name":     "John Doe",
		"scheduled_start": now.Format(time.RFC3339),
		"scheduled_end":   now.Add(-time.Hour).Format(time.RFC3339),
	})
	req := httptest.NewRequest(http.MethodPost, "/api/schedules", strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Scheduler")

	h := &ScheduleHandler{DB: nil}
	_ = h.CreateSchedule(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateScheduleInvalidID(t *testing.T) {
	e := echo.New()
	body := `{"client_name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/schedules/bad-id", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Scheduler")

	h := &ScheduleHandler{DB: nil}
	err := h.UpdateSchedule(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCancelScheduleInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/schedules/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Scheduler")

	h := &ScheduleHandler{DB: nil}
	err := h.CancelSchedule(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestConfirmAssignmentInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/schedules/bad-id/confirm", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Scheduler")

	h := &ScheduleHandler{DB: nil}
	err := h.ConfirmAssignment(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRequestBackupInvalidID(t *testing.T) {
	e := echo.New()
	body := `{"backup_staff_id":"` + uuid.New().String() + `","reason_code":"sick"}`
	req := httptest.NewRequest(http.MethodPost, "/api/schedules/bad-id/backup", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Scheduler")

	h := &ScheduleHandler{DB: nil}
	err := h.RequestBackup(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
