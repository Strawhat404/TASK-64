package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Governance handler tests ---
// Tests request validation for content governance endpoints.

func TestCreateContentInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/governance/content", strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Reviewer")

	h := &GovernanceHandler{DB: nil}
	err := h.CreateContent(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateContentMissingFields(t *testing.T) {
	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{
			"missing title",
			map[string]interface{}{"body": "Content body", "content_type": "article"},
		},
		{
			"missing body",
			map[string]interface{}{"title": "Title", "content_type": "article"},
		},
		{
			"missing content_type",
			map[string]interface{}{"title": "Title", "body": "Content body"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/governance/content", strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Reviewer")

			h := &GovernanceHandler{DB: nil}
			_ = h.CreateContent(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestCreateContentInvalidType(t *testing.T) {
	invalidTypes := []string{"blog", "page", "video", "", "Article"}

	for _, ct := range invalidTypes {
		t.Run("type_"+ct, func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(map[string]interface{}{
				"title":        "Test Title",
				"body":         "Test Body",
				"content_type": ct,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/governance/content", strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Reviewer")

			h := &GovernanceHandler{DB: nil}
			_ = h.CreateContent(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for type %q", rec.Code, http.StatusBadRequest, ct)
			}
		})
	}
}

func TestGetContentInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/governance/content/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Reviewer")

	h := &GovernanceHandler{DB: nil}
	err := h.GetContent(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestReviewDecisionInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/governance/reviews/"+uuid.New().String()+"/decide",
		strings.NewReader(`{broken`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	setTestUser(c, "Reviewer")

	h := &GovernanceHandler{DB: nil}
	err := h.ReviewDecision(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestReviewDecisionInvalidDecision(t *testing.T) {
	invalidDecisions := []string{"accept", "deny", "Approved", "REJECTED", ""}

	for _, d := range invalidDecisions {
		t.Run("decision_"+d, func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(map[string]string{"decision": d})
			req := httptest.NewRequest(http.MethodPost, "/api/governance/reviews/"+uuid.New().String()+"/decide",
				strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(uuid.New().String())

			setTestUser(c, "Reviewer")

			h := &GovernanceHandler{DB: nil}
			_ = h.ReviewDecision(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for decision %q", rec.Code, http.StatusBadRequest, d)
			}
		})
	}
}

func TestCreateRuleInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/governance/rules", strings.NewReader(`{bad`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Administrator")

	h := &GovernanceHandler{DB: nil}
	err := h.CreateRule(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateRuleInvalidType(t *testing.T) {
	invalidTypes := []string{"auto_block", "whitelist", "", "keyword"}

	for _, rt := range invalidTypes {
		t.Run("rule_type_"+rt, func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(map[string]string{
				"rule_type": rt,
				"pattern":   "test",
				"severity":  "low",
			})
			req := httptest.NewRequest(http.MethodPost, "/api/governance/rules", strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			setTestUser(c, "Administrator")

			h := &GovernanceHandler{DB: nil}
			_ = h.CreateRule(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for rule type %q", rec.Code, http.StatusBadRequest, rt)
			}
		})
	}
}

func TestCreateRelationshipInvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/governance/relationships", strings.NewReader(`{bad`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	setTestUser(c, "Reviewer")

	h := &GovernanceHandler{DB: nil}
	err := h.CreateRelationship(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// --- Governance versioning/diff/re-review/promote tests ---

func TestGetVersionHistoryInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/governance/content/bad-id/versions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Reviewer")

	h := &GovernanceHandler{DB: nil}
	_ = h.GetVersionHistory(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDiffVersionsInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/governance/content/bad-id/versions/diff?v1=1&v2=2", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Reviewer")

	h := &GovernanceHandler{DB: nil}
	_ = h.DiffVersions(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestPromoteContentInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/governance/content/bad-id/promote", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Administrator")

	h := &GovernanceHandler{DB: nil}
	_ = h.PromoteContent(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestReReviewInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/governance/content/bad-id/re-review", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Reviewer")

	h := &GovernanceHandler{DB: nil}
	_ = h.ReReview(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSubmitForReviewInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/governance/content/bad-id/submit", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Reviewer")

	h := &GovernanceHandler{DB: nil}
	_ = h.SubmitForReview(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateContentInvalidID(t *testing.T) {
	e := echo.New()
	body := `{"title":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/governance/content/bad-id", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Administrator")

	h := &GovernanceHandler{DB: nil}
	_ = h.UpdateContent(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateRuleInvalidID(t *testing.T) {
	e := echo.New()
	body := `{"severity":"critical"}`
	req := httptest.NewRequest(http.MethodPut, "/api/governance/rules/bad-id", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Administrator")

	h := &GovernanceHandler{DB: nil}
	_ = h.UpdateRule(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeleteRuleInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/governance/rules/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Administrator")

	h := &GovernanceHandler{DB: nil}
	_ = h.DeleteRule(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDeleteRelationshipInvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/governance/relationships/bad-id", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")
	setTestUser(c, "Administrator")

	h := &GovernanceHandler{DB: nil}
	_ = h.DeleteRelationship(c)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateRelationshipInvalidType(t *testing.T) {
	invalidTypes := []string{"related", "parent", "child", ""}
	for _, rt := range invalidTypes {
		t.Run("rel_type_"+rt, func(t *testing.T) {
			e := echo.New()
			b, _ := json.Marshal(map[string]string{
				"source_content_id":  uuid.New().String(),
				"target_content_id":  uuid.New().String(),
				"relationship_type": rt,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/governance/relationships", strings.NewReader(string(b)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			setTestUser(c, "Reviewer")

			h := &GovernanceHandler{DB: nil}
			_ = h.CreateRelationship(c)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for type %q", rec.Code, http.StatusBadRequest, rt)
			}
		})
	}
}

func TestRollbackContentInvalidID(t *testing.T) {
	e := echo.New()
	b, _ := json.Marshal(map[string]int{"target_version": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/governance/content/bad-id/rollback", strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	setTestUser(c, "Administrator")

	h := &GovernanceHandler{DB: nil}
	err := h.RollbackContent(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
