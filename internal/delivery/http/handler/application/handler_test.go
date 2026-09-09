package application

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appRepo "github.com/itauq-golang/internal/repository/application"
	appUc "github.com/itauq-golang/internal/usecase/application"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := appRepo.NewMemoryRepository()
	uc := appUc.NewUsecase(repo, appUc.MemoryProvisioner{})

	r := gin.New()
	h := NewHandler(uc)
	r.POST("/applications", h.Create)
	r.GET("/applications", h.List)
	r.GET("/applications/:id", h.Get)
	r.POST("/applications/:id/approve", h.Approve)
	r.POST("/applications/:id/reject", h.Reject)
	return r
}

func createApplication(t *testing.T, r *gin.Engine) string {
	t.Helper()
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"Jane Doe","email":"jane@example.com","institution":"ITB","occupation":"Student","reason":"Research"}`)
	req, _ := http.NewRequest(http.MethodPost, "/applications", body)
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
	body := bytes.NewBufferString(`{"full_name":"John Doe","email":"john@example.com","institution":"UI","occupation":"Researcher","reason":"Testing"}`)
	req, _ := http.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || resp.Data.Status != "pending" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreateRejectsMissingName(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"email":"x@y.com"}`)
	req, _ := http.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRejectsInvalidEmail(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"John","email":"not-an-email"}`)
	req, _ := http.NewRequest(http.MethodPost, "/applications", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestListHappy(t *testing.T) {
	r := newTestRouter(t)
	createApplication(t, r)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/applications", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestGetHappy(t *testing.T) {
	r := newTestRouter(t)
	id := createApplication(t, r)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/applications/"+id, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestGetNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/applications/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestApproveHappy(t *testing.T) {
	r := newTestRouter(t)
	id := createApplication(t, r)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"review_note":"Approved"}`)
	req, _ := http.NewRequest(http.MethodPost, "/applications/"+id+"/approve", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "super-admin")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestApproveNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/applications/nonexistent/approve", nil)
	req.Header.Set("X-User-ID", "super-admin")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestRejectHappy(t *testing.T) {
	r := newTestRouter(t)
	id := createApplication(t, r)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"review_note":"Rejected"}`)
	req, _ := http.NewRequest(http.MethodPost, "/applications/"+id+"/reject", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "super-admin")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestRejectNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"review_note":"nope"}`)
	req, _ := http.NewRequest(http.MethodPost, "/applications/nonexistent/reject", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "super-admin")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}
