package profile

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	pRepo "github.com/itauq-golang/internal/repository/profile"
	pUc "github.com/itauq-golang/internal/usecase/profile"

	"github.com/gin-gonic/gin"
)

const userID = "user-1"

func TestGetMeNoUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := pRepo.NewMemoryRepository()
	uc := pUc.NewUsecase(repo, pUc.MemoryPasswordUpdater{})

	r := gin.New()
	h := NewHandler(uc)
	r.GET("/profile/me", h.GetMe)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/profile/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401. body=%s", w.Code, w.Body.String())
	}
}

func TestGetMeNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := pRepo.NewMemoryRepository()
	uc := pUc.NewUsecase(repo, pUc.MemoryPasswordUpdater{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "nonexistent")
		c.Next()
	})
	h := NewHandler(uc)
	r.GET("/profile/me", h.GetMe)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/profile/me", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateMeNoUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := pRepo.NewMemoryRepository()
	uc := pUc.NewUsecase(repo, pUc.MemoryPasswordUpdater{})

	r := gin.New()
	h := NewHandler(uc)
	r.PUT("/profile/me", h.UpdateMe)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"Test"}`)
	req, _ := http.NewRequest(http.MethodPut, "/profile/me", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateMeNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := pRepo.NewMemoryRepository()
	uc := pUc.NewUsecase(repo, pUc.MemoryPasswordUpdater{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "nonexistent")
		c.Next()
	})
	h := NewHandler(uc)
	r.PUT("/profile/me", h.UpdateMe)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"Test"}`)
	req, _ := http.NewRequest(http.MethodPut, "/profile/me", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestChangePasswordNoUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := pRepo.NewMemoryRepository()
	uc := pUc.NewUsecase(repo, pUc.MemoryPasswordUpdater{})

	r := gin.New()
	h := NewHandler(uc)
	r.PUT("/profile/me/password", h.ChangePassword)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"current_password":"old12345","new_password":"new123456"}`)
	req, _ := http.NewRequest(http.MethodPut, "/profile/me/password", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401. body=%s", w.Code, w.Body.String())
	}
}

func TestChangePasswordRejectsShortPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := pRepo.NewMemoryRepository()
	uc := pUc.NewUsecase(repo, pUc.MemoryPasswordUpdater{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	h := NewHandler(uc)
	r.PUT("/profile/me/password", h.ChangePassword)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"current_password":"old","new_password":"short"}`)
	req, _ := http.NewRequest(http.MethodPut, "/profile/me/password", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}
