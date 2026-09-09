package taskscenario

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qDomain "github.com/itauq-golang/internal/domain/questionnaire"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	tsRepo "github.com/itauq-golang/internal/repository/taskscenario"
	tsUc "github.com/itauq-golang/internal/usecase/taskscenario"

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

	tr := tsRepo.NewMemoryRepository()
	uc := tsUc.NewUsecase(tr, qr)

	r := gin.New()
	g := r.Group("/questionnaires/:id/task-scenarios")
	h := NewHandler(uc)
	g.POST("", h.Create)
	g.GET("", h.List)

	g2 := r.Group("/task-scenarios")
	g2.GET("/:id", h.Get)
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
	body := bytes.NewBufferString(`{"title":"Task 1","instruction":"Do this","task_order":1}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/task-scenarios", body)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || resp.Data.Title != "Task 1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreateRejectsMissingTitle(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"instruction":"Do this"}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/task-scenarios", body)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestCreateForbiddenForNonOwner(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"Task","instruction":"Do this"}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/task-scenarios", body)
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

	// seed a task scenario
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"Task 1","instruction":"Do this","task_order":1}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/task-scenarios", body)
	withCaller(req)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("seed: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/questionnaires/"+questionnaireID+"/task-scenarios", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 task scenario, got %d", len(resp.Data))
	}
}

func TestGetNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/task-scenarios/nonexistent", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateHappy(t *testing.T) {
	r := newTestRouter(t)

	// create task scenario
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"Original","instruction":"Do this","task_order":1}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/task-scenarios", body)
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

	// update
	w = httptest.NewRecorder()
	body = bytes.NewBufferString(`{"title":"Updated Title"}`)
	req, _ = http.NewRequest(http.MethodPut, "/task-scenarios/"+created.Data.ID, body)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteHappy(t *testing.T) {
	r := newTestRouter(t)

	// create task scenario
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"To delete","instruction":"Do this","task_order":1}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires/"+questionnaireID+"/task-scenarios", body)
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
	req, _ = http.NewRequest(http.MethodDelete, "/task-scenarios/"+created.Data.ID, nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/task-scenarios/nonexistent", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}
