package application

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	domain "github.com/itauq-golang/internal/domain/application"
	repo "github.com/itauq-golang/internal/repository/application"
	u "github.com/itauq-golang/internal/usecase/application"
)

type Handler struct{ uc *u.Usecase }

func NewHandler(uc *u.Usecase) *Handler { return &Handler{uc: uc} }
func ok(c *gin.Context, data interface{}, meta interface{}) {
	body := gin.H{"success": true, "data": data}
	if meta != nil {
		body["meta"] = meta
	}
	c.JSON(http.StatusOK, body)
}
func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"success": false, "error": gin.H{"code": code, "message": message}})
}
func (h *Handler) Create(c *gin.Context) {
	var in domain.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	a, err := h.uc.Create(c.Request.Context(), in)
	if err != nil {
		if errors.Is(err, repo.ErrDuplicatePending) {
			fail(c, 409, "DUPLICATE_APPLICATION", err.Error())
			return
		}
		fail(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	c.JSON(201, gin.H{"success": true, "data": gin.H{"id": a.ID, "status": a.Status, "created_at": a.CreatedAt}})
}
func (h *Handler) List(c *gin.Context) {
	page := parseInt(c.Query("page"), 1)
	size := parseInt(c.Query("page_size"), 20)
	items, meta, err := h.uc.List(c.Request.Context(), domain.Status(c.Query("status")), page, size)
	if err != nil {
		fail(c, 500, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, items, meta)
}
func (h *Handler) Get(c *gin.Context) {
	a, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		fail(c, 404, "NOT_FOUND", "application not found")
		return
	}
	ok(c, a, nil)
}
func (h *Handler) Approve(c *gin.Context) {
	var in domain.ReviewInput
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&in); err != nil {
			fail(c, 400, "VALIDATION_ERROR", err.Error())
			return
		}
	}
	result, err := h.uc.Approve(c.Request.Context(), c.Param("id"), in.ReviewNote, c.GetHeader("X-User-ID"))
	if err != nil {
		h.reviewError(c, err)
		return
	}
	ok(c, result, nil)
}
func (h *Handler) Reject(c *gin.Context) {
	var in domain.ReviewInput
	if err := c.ShouldBindJSON(&in); err != nil && err.Error() != "EOF" {
		fail(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	a, err := h.uc.Reject(c.Request.Context(), c.Param("id"), in.ReviewNote, c.GetHeader("X-User-ID"))
	if err != nil {
		h.reviewError(c, err)
		return
	}
	ok(c, a, nil)
}
func (h *Handler) reviewError(c *gin.Context, err error) {
	if errors.Is(err, repo.ErrNotFound) {
		fail(c, 404, "NOT_FOUND", "application not found")
		return
	}
	if errors.Is(err, repo.ErrDuplicatePending) {
		fail(c, 409, "APPLICATION_ALREADY_REVIEWED", "application is not pending")
		return
	}
	fail(c, 500, "INTERNAL_ERROR", err.Error())
}
func parseInt(v string, d int) int {
	if n, e := strconv.Atoi(v); e == nil && n > 0 {
		return n
	}
	return d
}
