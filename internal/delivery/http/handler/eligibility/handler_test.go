package eligibility

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qDomain "github.com/itauq-golang/internal/domain/questionnaire"
	eligRepo "github.com/itauq-golang/internal/repository/eligibility"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	eligUc "github.com/itauq-golang/internal/usecase/eligibility"

	"github.com/gin-gonic/gin"
)

const (
	adminID        = "admin-1"
	questionnaireID = "q-1"
)

func newTestRouter(t *testing.T) (*gin.Engine, *qRepo.MemoryRepository) {
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

	er := eligRepo.NewMemoryRepository()
	uc := eligUc.NewUsecase(er, qr)

	r := gin.New()
	g := r.Group("/questionnaires/:id/eligibility")
	h := NewHandler(uc)
	g.POST("", h.Create)
	g.GET("", h.List)

	g2 := r.Group("/eligibility")
	g2.GET("/:id", h.Get)
	g2.PUT("/:id", h.Update)
	g2.DELETE("/:id", h.Delete)

	return r, qr
}

func withCaller(req *http.Request) {
	req.Header.Set("X-User-ID", adminID)
	req.Header.Set("X-User-Role", "administrator")
	req.Header.Set("Content-Type", "application/json")
}

func TestCreateHappy(t *testing.T) {
	r, _ := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"statement":"Must be 18+","criteria_order":1}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/eligibility", body)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201. body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRejectsEmptyBody(t *testing.T) {
	r, _ := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/eligibility", body)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestCreateForbiddenForNonOwner(t *testing.T) {
	r, _ := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"statement":"test"}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/eligibility", body)
	req.Header.Set("X-User-ID", "other-admin")
	req.Header.Set("X-User-Role", "administrator")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403. body=%s", w.Code, w.Body.String())
	}
}

func TestListHappy(t *testing.T) {
	r, _ := newTestRouter(t)

	// seed a criterion
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"statement":"Must be 18+","criteria_order":1}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/eligibility", body)
	withCaller(req)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("seed: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/questionnaires/"+questionnaireID+"/eligibility", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestGetNotFound(t *testing.T) {
	r, _ := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/eligibility/nonexistent", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateHappy(t *testing.T) {
	r, _ := newTestRouter(t)

	// create criterion
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"statement":"Original","criteria_order":1}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/eligibility", body)
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

	// update it
	w = httptest.NewRecorder()
	body = bytes.NewBufferString(`{"statement":"Updated"}`)
	req, _ = http.NewRequest(http.MethodPut, "/eligibility/"+created.Data.ID, body)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteHappy(t *testing.T) {
	r, _ := newTestRouter(t)

	// create criterion
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"statement":"To delete","criteria_order":1}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/eligibility", body)
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

	// delete it
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, "/eligibility/"+created.Data.ID, nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204. body=%s", w.Code, w.Body.String())
	}
}
