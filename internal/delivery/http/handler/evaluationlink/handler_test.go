package evaluationlink

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qDomain "github.com/itauq-golang/internal/domain/questionnaire"
	elRepo "github.com/itauq-golang/internal/repository/evaluationlink"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	elUc "github.com/itauq-golang/internal/usecase/evaluationlink"

	"github.com/gin-gonic/gin"
)

const (
	adminID        = "admin-1"
	questionnaireID = "q-1"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	qr := qRepo.NewMemoryRepository()
	now := time.Now().UTC()
	if err := qr.Create(context.Background(), &qDomain.Questionnaire{
		ID: questionnaireID, AdministratorID: adminID,
		Title: "Test", AppName: "App", Status: qDomain.StatusActive,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed questionnaire: %v", err)
	}

	er := elRepo.NewMemoryRepository()
	uc := elUc.NewUsecase(er, qr, "https://itauq.site/e/")

	r := gin.New()
	g := r.Group("/questionnaires/:id/evaluation-links")
	h := NewHandler(uc)
	g.POST("", h.Create)
	g.GET("", h.List)

	g2 := r.Group("/evaluation-links")
	g2.PUT("/:id", h.Update)
	g2.DELETE("/:id", h.Delete)

	return r
}

func withCaller(req *http.Request) {
	req.Header.Set("X-User-ID", adminID)
	req.Header.Set("X-User-Role", "administrator")
	req.Header.Set("Content-Type", "application/json")
}

func TestCreateHappy(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/evaluation-links", body)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string `json:"token"`
			URL   string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data.Token) != 8 {
		t.Fatalf("expected 8-char token, got %q", resp.Data.Token)
	}
}

func TestCreateForbiddenForNonOwner(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/evaluation-links", body)
	req.Header.Set("X-User-ID", "other-admin")
	req.Header.Set("X-User-Role", "administrator")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403. body=%s", w.Code, w.Body.String())
	}
}

func TestListHappy(t *testing.T) {
	r := newTestRouter(t)

	// seed a link
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/evaluation-links", body)
	withCaller(req)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("seed: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/questionnaires/"+questionnaireID+"/evaluation-links", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateHappy(t *testing.T) {
	r := newTestRouter(t)

	// create link
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/evaluation-links", body)
	withCaller(req)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// deactivate
	w = httptest.NewRecorder()
	body = bytes.NewBufferString(`{"is_active":false}`)
	req, _ = http.NewRequest(http.MethodPut, "/evaluation-links/"+created.Data.ID, body)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteHappy(t *testing.T) {
	r := newTestRouter(t)

	// create link
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/evaluation-links", body)
	withCaller(req)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// delete
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, "/evaluation-links/"+created.Data.ID, nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/evaluation-links/nonexistent", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}
