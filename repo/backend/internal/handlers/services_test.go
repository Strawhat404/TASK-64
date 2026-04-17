package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Service handler tests ---
// Tests request validation, tier/duration/headcount constraints, and parameter parsing.

func TestCreateServiceInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	err := h.CreateService(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateServiceInvalidTier(t *testing.T) {
	invalidTiers := []string{"basic", "pro", "free", "", "Standard"}

	for _, tier := range invalidTiers {
		t.Run("tier_"+tier, func(t *testing.T) {
			e := echo.New()
			body := `{"name":"Test Service","tier":"` + tier + `","duration_minutes":60,"headcount":1,"base_price_usd":50}`
			req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Administrator")

			h := &ServiceHandler{DB: nil}
			_ = h.CreateService(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for tier %q", rec.Code, http.StatusBadRequest, tier)
			}
		})
	}
}

func TestCreateServiceInvalidDuration(t *testing.T) {
	invalidDurations := []int{0, 10, 14, 16, 20, 241, 300}

	for _, dur := range invalidDurations {
		t.Run(fmt.Sprintf("duration_%d", dur), func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(map[string]interface{}{
				"name":             "Test",
				"tier":             "standard",
				"duration_minutes": dur,
				"headcount":        1,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Administrator")

			h := &ServiceHandler{DB: nil}
			_ = h.CreateService(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for duration %d", rec.Code, http.StatusBadRequest, dur)
			}
		})
	}
}

func TestCreateServiceInvalidHeadcount(t *testing.T) {
	invalidHeadcounts := []int{0, -1, 11, 100}

	for _, hc := range invalidHeadcounts {
		e := echo.New()
		b, _ := json.Marshal(map[string]interface{}{
			"name":             "Test",
			"tier":             "standard",
			"duration_minutes": 60,
			"headcount":        hc,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(string(b)))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setTestUser(c, "Administrator")

		h := &ServiceHandler{DB: nil}
		_ = h.CreateService(c)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d for headcount %d", rec.Code, http.StatusBadRequest, hc)
		}
	}
}

func TestCreateServiceMissingName(t *testing.T) {
	e := echo.New()
	body := `{"tier":"standard","duration_minutes":60,"headcount":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	_ = h.CreateService(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateServiceValidTiers(t *testing.T) {
	validTiers := []string{"standard", "premium", "enterprise"}

	for _, tier := range validTiers {
		t.Run(tier, func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(map[string]interface{}{
				"name":             "Test Service",
				"tier":             tier,
				"duration_minutes": 60,
				"headcount":        1,
				"base_price_usd":  100,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Administrator")

			h := &ServiceHandler{DB: nil}
			_ = h.CreateService(c)
			// Without a DB, we'll get an internal error (nil pointer), but NOT a validation error
			// The key check is that it passes validation and does not return 400
			if rec.Code == http.StatusBadRequest {
				t.Errorf("tier %q should pass validation", tier)
			}
		})
	}
}

func TestGetServiceInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/services/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")

	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	err := h.GetService(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeleteServiceInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/services/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	err := h.DeleteService(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetPricingInvalidServiceID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/services/bad-id/pricing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	err := h.GetPricing(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateServiceInvalidTier(t *testing.T) {
	e := echo.New()
	b, _ := json.Marshal(map[string]interface{}{
		"tier": "invalid",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/services/"+uuid.New().String(), strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	_ = h.UpdateService(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateServiceInvalidDuration(t *testing.T) {
	e := echo.New()
	b, _ := json.Marshal(map[string]interface{}{
		"duration_minutes": 17,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/services/"+uuid.New().String(), strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	_ = h.UpdateService(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// --- Response contract tests ---

func TestCreateService_ErrorResponseContract(t *testing.T) {
	e := echo.New()
	// Invalid tier
	body := `{"name":"Test","tier":"invalid","duration_minutes":60,"headcount":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	_ = h.CreateService(c)

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	errMsg, ok := resp["error"].(string)
	if !ok || errMsg == "" {
		t.Error("error response must have non-empty 'error' string field")
	}
	if errMsg != "Tier must be one of: standard, premium, enterprise" {
		t.Errorf("error = %q, want tier validation message", errMsg)
	}
}

func TestGetService_InvalidID_ErrorContract(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/services/not-valid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-valid")
	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	_ = h.GetService(c)

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("400 response is not valid JSON: %v", err)
	}
	if resp["error"] != "Invalid service ID" {
		t.Errorf("error = %q, want %q", resp["error"], "Invalid service ID")
	}
}

func TestCreateService_DurationValidation_ErrorContract(t *testing.T) {
	e := echo.New()
	body := `{"name":"Test","tier":"standard","duration_minutes":17,"headcount":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	_ = h.CreateService(c)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	errMsg := resp["error"].(string)
	if errMsg != "Duration must be between 15 and 240 minutes in 15-minute increments" {
		t.Errorf("error = %q, want duration validation message", errMsg)
	}
}

func TestCreateService_HeadcountValidation_ErrorContract(t *testing.T) {
	e := echo.New()
	body := `{"name":"Test","tier":"standard","duration_minutes":60,"headcount":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/services", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	_ = h.CreateService(c)

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	errMsg := resp["error"].(string)
	if errMsg != "Headcount must be between 1 and 10" {
		t.Errorf("error = %q, want headcount validation message", errMsg)
	}
}

func TestUpdateServiceInvalidHeadcount(t *testing.T) {
	e := echo.New()
	b, _ := json.Marshal(map[string]interface{}{
		"headcount": 15,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/services/"+uuid.New().String(), strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	setTestUser(c, "Administrator")

	h := &ServiceHandler{DB: nil}
	_ = h.UpdateService(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
