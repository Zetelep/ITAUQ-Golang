package questionnaire

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	qUc "github.com/itauq-golang/internal/usecase/questionnaire"

	"github.com/gin-gonic/gin"
)

const adminID = "admin-1"

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := qRepo.NewMemoryRepository()
	uc := qUc.NewUsecase(repo)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", adminID)
		c.Set("user_role", "administrator")
		c.Next()
	})
	h := NewHandler(uc)
	r.POST("/questionnaires", h.Create)
	r.GET("/questionnaires", h.List)
	r.GET("/questionnaires/:id", h.Get)
	r.PUT("/questionnaires/:id", h.Update)
	r.DELETE("/questionnaires/:id", h.Delete)
	return r
}

func createQuestionnaire(t *testing.T, r *gin.Engine) string {
	t.Helper()
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"Test Q","app_name":"TestApp","description":"desc"}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("seed: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return resp.Data.ID
}

func TestCreateHappy(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"New Q","app_name":"App"}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201. body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRejectsMissingTitle(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"app_name":"App"}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRejectsNoUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := qRepo.NewMemoryRepository()
	uc := qUc.NewUsecase(repo)

	r := gin.New()
	h := NewHandler(uc)
	r.POST("/questionnaires", h.Create)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"Q","app_name":"App"}`)
	req, _ := http.NewRequest(http.MethodPost, "/questionnaires", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401. body=%s", w.Code, w.Body.String())
	}
}

func TestListHappy(t *testing.T) {
	r := newTestRouter(t)
	createQuestionnaire(t, r)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/questionnaires", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []any `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Meta.Total != 1 {
		t.Fatalf("expected 1 questionnaire, got %d", resp.Meta.Total)
	}
}

func TestGetHappy(t *testing.T) {
	r := newTestRouter(t)
	id := createQuestionnaire(t, r)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/questionnaires/"+id, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestGetNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/questionnaires/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateHappy(t *testing.T) {
	r := newTestRouter(t)
	id := createQuestionnaire(t, r)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"Updated Title"}`)
	req, _ := http.NewRequest(http.MethodPut, "/questionnaires/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"title":"X"}`)
	req, _ := http.NewRequest(http.MethodPut, "/questionnaires/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteHappy(t *testing.T) {
	r := newTestRouter(t)
	id := createQuestionnaire(t, r)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/questionnaires/"+id, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/questionnaires/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}
