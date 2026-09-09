package profile

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	domain "github.com/itauq-golang/internal/domain/profile"
	repo "github.com/itauq-golang/internal/repository/profile"
	u "github.com/itauq-golang/internal/usecase/profile"
)

type Handler struct{ uc *u.Usecase }

func NewHandler(uc *u.Usecase) *Handler { return &Handler{uc: uc} }

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"success": false, "error": gin.H{"code": code, "message": message}})
}

func (h *Handler) GetMe(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	p, err := h.uc.GetMe(c.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			fail(c, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, p)
}

func (h *Handler) UpdateMe(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	var in domain.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	p, err := h.uc.UpdateMe(c.Request.Context(), uid, in)
	if err != nil {
		if errors.Is(err, u.ErrFullNameRequired) {
			fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		if errors.Is(err, repo.ErrNotFound) {
			fail(c, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, p)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	uid := c.GetString("user_id")
	if uid == "" {
		fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.uc.ChangePassword(c.Request.Context(), uid, in.CurrentPassword, in.NewPassword); err != nil {
		fail(c, http.StatusBadRequest, "PASSWORD_UPDATE_FAILED", err.Error())
		return
	}
	ok(c, gin.H{"message": "password updated"})
}
