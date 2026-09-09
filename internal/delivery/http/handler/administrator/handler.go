package administrator

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	domain "github.com/itauq-golang/internal/domain/administrator"
	repo "github.com/itauq-golang/internal/repository/administrator"
	u "github.com/itauq-golang/internal/usecase/administrator"
)

type Handler struct{ uc *u.Usecase }

func NewHandler(uc *u.Usecase) *Handler { return &Handler{uc: uc} }

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
func okWithMeta(c *gin.Context, data any, meta any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "meta": meta})
}
func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"success": false, "error": gin.H{"code": code, "message": message}})
}

func (h *Handler) List(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	size := parseInt(c.Query("page_size"), 20)
	filter := domain.ListFilter{Page: page, PageSize: size}
	if raw := c.Query("is_active"); raw != "" {
		if b, err := strconv.ParseBool(raw); err == nil {
			filter.IsActive = &b
		}
	}
	items, meta, err := h.uc.List(c.Request.Context(), filter)
	if err != nil {
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	okWithMeta(c, items, meta)
}

func (h *Handler) Get(c *gin.Context) {
	a, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			fail(c, http.StatusNotFound, "NOT_FOUND", "administrator not found")
			return
		}
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, a)
}

func (h *Handler) Create(c *gin.Context) {
	var in domain.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	a, err := h.uc.Create(c.Request.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, u.ErrFullNameRequired), errors.Is(err, u.ErrEmailRequired):
			fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		default:
			fail(c, http.StatusBadGateway, "INVITE_FAILED", err.Error())
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": a})
}

func (h *Handler) Update(c *gin.Context) {
	var in domain.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	a, err := h.uc.Update(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		switch {
		case errors.Is(err, u.ErrFullNameRequired):
			fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		case errors.Is(err, repo.ErrNotFound):
			fail(c, http.StatusNotFound, "NOT_FOUND", "administrator not found")
		default:
			fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	ok(c, a)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.uc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			fail(c, http.StatusNotFound, "NOT_FOUND", "administrator not found")
			return
		}
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func parseInt(v string, d int) int {
	if n, e := strconv.Atoi(v); e == nil && n > 0 {
		return n
	}
	return d
}
