package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "github.com/itauq-golang/internal/domain/profile"
	pRepo "github.com/itauq-golang/internal/repository/profile"
	pUc "github.com/itauq-golang/internal/usecase/profile"

	"github.com/gin-gonic/gin"
)

const userID = "user-1"

type updateProfileRepository struct {
	profile domain.Profile
}

func (r *updateProfileRepository) GetByID(_ context.Context, _ string) (*domain.Profile, error) {
	profile := r.profile
	return &profile, nil
}

func (r *updateProfileRepository) Update(_ context.Context, _ string, in domain.UpdateInput) (*domain.Profile, error) {
	if in.FullName != nil {
		r.profile.FullName = *in.FullName
	}
	if in.Institution != nil {
		r.profile.Institution = *in.Institution
	}
	if in.Occupation != nil {
		r.profile.Occupation = *in.Occupation
	}
	profile := r.profile
	return &profile, nil
}

func (r *updateProfileRepository) ClearMustChangePassword(context.Context, string) error {
	return nil
}

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
	r.PATCH("/profiles/me", h.UpdateMe)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"Test"}`)
	req, _ := http.NewRequest(http.MethodPatch, "/profiles/me", body)
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
	r.PATCH("/profiles/me", h.UpdateMe)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"full_name":"Test"}`)
	req, _ := http.NewRequest(http.MethodPatch, "/profiles/me", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateMeInstitutionAndOccupation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &updateProfileRepository{profile: domain.Profile{
		ID:          userID,
		FullName:    "Existing Name",
		Institution: "Old Institution",
		Occupation:  "Old Occupation",
	}}
	uc := pUc.NewUsecase(repo, pUc.MemoryPasswordUpdater{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	h := NewHandler(uc)
	r.PATCH("/profiles/me", h.UpdateMe)

	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"institution":"  Universitas ABC  ","occupation":"  Dosen  "}`)
	req, _ := http.NewRequest(http.MethodPatch, "/profiles/me", body)
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
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.FullName != "Existing Name" {
		t.Fatalf("expected full_name preserved, got %q", resp.Data.FullName)
	}
	if resp.Data.Institution != "Universitas ABC" {
		t.Fatalf("expected institution trimmed, got %q", resp.Data.Institution)
	}
	if resp.Data.Occupation != "Dosen" {
		t.Fatalf("expected occupation trimmed, got %q", resp.Data.Occupation)
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
