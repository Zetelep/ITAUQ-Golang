package questionnaire

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	domain "github.com/itauq-golang/internal/domain/questionnaire"
	repo "github.com/itauq-golang/internal/repository/questionnaire"
	u "github.com/itauq-golang/internal/usecase/questionnaire"
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

func getCallerID(c *gin.Context) string {
	if id, exists := c.Get("user_id"); exists {
		return id.(string)
	}
	return c.GetHeader("X-User-ID")
}

func getCallerRole(c *gin.Context) string {
	if role, exists := c.Get("user_role"); exists {
		return role.(string)
	}
	return c.GetHeader("X-User-Role")
}

func (h *Handler) Create(c *gin.Context) {
	var in domain.CreateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	callerID := getCallerID(c)
	if callerID == "" {
		fail(c, 401, "UNAUTHORIZED", "user identity not found")
		return
	}
	q, err := h.uc.Create(c.Request.Context(), in, callerID)
	if err != nil {
		fail(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	c.JSON(201, gin.H{"success": true, "data": q})
}

func (h *Handler) List(c *gin.Context) {
	callerID := getCallerID(c)
	callerRole := getCallerRole(c)
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)
	status := domain.Status(c.Query("status"))
	administratorID := c.Query("administrator_id")

	items, meta, err := h.uc.List(c.Request.Context(), callerID, callerRole, status, administratorID, page, pageSize)
	if err != nil {
		fail(c, 500, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, items, meta)
}

func (h *Handler) Get(c *gin.Context) {
	callerID := getCallerID(c)
	callerRole := getCallerRole(c)
	q, err := h.uc.Get(c.Request.Context(), c.Param("id"), callerID, callerRole)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			fail(c, 404, "NOT_FOUND", "questionnaire not found")
			return
		}
		if errors.Is(err, repo.ErrForbidden) {
			fail(c, 403, "FORBIDDEN", "not authorized to access this questionnaire")
			return
		}
		fail(c, 500, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, q, nil)
}

func (h *Handler) Update(c *gin.Context) {
	var in domain.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	callerID := getCallerID(c)
	if callerID == "" {
		fail(c, 401, "UNAUTHORIZED", "user identity not found")
		return
	}
	q, err := h.uc.Update(c.Request.Context(), c.Param("id"), in, callerID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			fail(c, 404, "NOT_FOUND", "questionnaire not found")
			return
		}
		if errors.Is(err, repo.ErrForbidden) {
			fail(c, 403, "FORBIDDEN", "not authorized to update this questionnaire")
			return
		}
		fail(c, 500, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, q, nil)
}

func (h *Handler) Delete(c *gin.Context) {
	callerID := getCallerID(c)
	if callerID == "" {
		fail(c, 401, "UNAUTHORIZED", "user identity not found")
		return
	}
	err := h.uc.Delete(c.Request.Context(), c.Param("id"), callerID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			fail(c, 404, "NOT_FOUND", "questionnaire not found")
			return
		}
		if errors.Is(err, repo.ErrForbidden) {
			fail(c, 403, "FORBIDDEN", "not authorized to delete this questionnaire")
			return
		}
		fail(c, 500, "INTERNAL_ERROR", err.Error())
		return
	}
	c.JSON(204, nil)
}

func parseInt(v string, d int) int {
	if n, e := strconv.Atoi(v); e == nil && n > 0 {
		return n
	}
	return d
}
