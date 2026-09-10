package administrator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	aRepo "github.com/itauq-golang/internal/repository/administrator"
	aUsecase "github.com/itauq-golang/internal/usecase/administrator"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := aRepo.NewMemoryRepository()
	uc := aUsecase.NewUsecase(repo, aUsecase.MemoryInviter{})

	r := gin.New()
	g := r.Group("/administrators")
	h := NewHandler(uc)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	return r
}

func seedAdmin(t *testing.T, r *gin.Engine) string {
	t.Helper()
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"John Doe","email":"john@example.com"}`)
	req, _ := http.NewRequest(http.MethodPost, "/administrators", body)
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
	body := bytes.NewBufferString(`{"full_name":"Jane Doe","email":"jane@example.com"}`)
	req, _ := http.NewRequest(http.MethodPost, "/administrators", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			FullName string `json:"full_name"`
			Email    string `json:"email"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || resp.Data.FullName != "Jane Doe" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreateRejectsEmptyFullName(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"","email":"x@y.com"}`)
	req, _ := http.NewRequest(http.MethodPost, "/administrators", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestCreateRejectsMissingEmail(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"Jane"}`)
	req, _ := http.NewRequest(http.MethodPost, "/administrators", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestListReturnsSeededAdmin(t *testing.T) {
	r := newTestRouter(t)
	seedAdmin(t, r)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/administrators", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []struct {
			FullName string `json:"full_name"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Meta.Total != 1 {
		t.Fatalf("expected 1 admin, got %d", resp.Meta.Total)
	}
}

func TestGetHappy(t *testing.T) {
	r := newTestRouter(t)
	id := seedAdmin(t, r)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/administrators/"+id, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
}

func TestGetNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/administrators/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateHappy(t *testing.T) {
	r := newTestRouter(t)
	id := seedAdmin(t, r)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"Updated Name"}`)
	req, _ := http.NewRequest(http.MethodPatch, "/administrators/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			FullName string `json:"full_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.FullName != "Updated Name" {
		t.Fatalf("expected updated name, got %s", resp.Data.FullName)
	}
}

func TestUpdateNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"X"}`)
	req, _ := http.NewRequest(http.MethodPatch, "/administrators/nonexistent", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteHappy(t *testing.T) {
	r := newTestRouter(t)
	id := seedAdmin(t, r)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/administrators/"+id, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: got %d, want 204. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateIsActiveOnly(t *testing.T) {
	r := newTestRouter(t)
	id := seedAdmin(t, r)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"is_active":false}`)
	req, _ := http.NewRequest(http.MethodPatch, "/administrators/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			FullName string `json:"full_name"`
			IsActive bool   `json:"is_active"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.IsActive {
		t.Fatalf("expected is_active=false, got true")
	}
	if resp.Data.FullName != "John Doe" {
		t.Fatalf("expected full_name preserved, got %s", resp.Data.FullName)
	}
}

func TestUpdateBothFields(t *testing.T) {
	r := newTestRouter(t)
	id := seedAdmin(t, r)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"Jane Smith","institution":"  Updated University  ","occupation":"  Researcher  ","is_active":false}`)
	req, _ := http.NewRequest(http.MethodPatch, "/administrators/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			FullName    string `json:"full_name"`
			Institution string `json:"institution"`
			Occupation  string `json:"occupation"`
			IsActive    bool   `json:"is_active"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.FullName != "Jane Smith" {
		t.Fatalf("expected full_name=Jane Smith, got %s", resp.Data.FullName)
	}

	if resp.Data.IsActive {
		t.Fatalf("expected is_active=false, got true")
	}
	if resp.Data.Institution != "Updated University" {
		t.Fatalf("expected trimmed institution, got %q", resp.Data.Institution)
	}
	if resp.Data.Occupation != "Researcher" {
		t.Fatalf("expected trimmed occupation, got %q", resp.Data.Occupation)
	}
}

func TestUpdateRejectsEmptyFullName(t *testing.T) {
	r := newTestRouter(t)
	id := seedAdmin(t, r)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"   "}`)
	req, _ := http.NewRequest(http.MethodPatch, "/administrators/"+id, body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, "/administrators/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}
