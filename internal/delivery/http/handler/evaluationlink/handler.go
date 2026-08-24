package evaluationlink

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	domain "github.com/itauq-golang/internal/domain/evaluationlink"
	repo "github.com/itauq-golang/internal/repository/evaluationlink"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	u "github.com/itauq-golang/internal/usecase/evaluationlink"
)

type Handler struct{ uc *u.Usecase }

func NewHandler(uc *u.Usecase) *Handler { return &Handler{uc: uc} }

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
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
	callerRole := getCallerRole(c)
	questionnaireID := c.Param("id")

	l, err := h.uc.Create(c.Request.Context(), questionnaireID, in, callerID, callerRole)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(201, gin.H{"success": true, "data": l})
}

func (h *Handler) List(c *gin.Context) {
	callerID := getCallerID(c)
	callerRole := getCallerRole(c)
	questionnaireID := c.Param("id")

	items, err := h.uc.ListByQuestionnaire(c.Request.Context(), questionnaireID, callerID, callerRole)
	if err != nil {
		handleError(c, err)
		return
	}
	ok(c, items)
}

func (h *Handler) Update(c *gin.Context) {
	var in domain.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	callerID := getCallerID(c)

	l, err := h.uc.Update(c.Request.Context(), c.Param("id"), in, callerID)
	if err != nil {
		handleError(c, err)
		return
	}
	ok(c, l)
}

func (h *Handler) Delete(c *gin.Context) {
	callerID := getCallerID(c)

	err := h.uc.Delete(c.Request.Context(), c.Param("id"), callerID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(204, nil)
}

func handleError(c *gin.Context, err error) {
	if errors.Is(err, repo.ErrNotFound) || errors.Is(err, qRepo.ErrNotFound) {
		fail(c, 404, "NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, qRepo.ErrForbidden) {
		fail(c, 403, "FORBIDDEN", "not authorized to access this resource")
		return
	}
	if errors.Is(err, u.ErrInvalidExpiry) {
		fail(c, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	fail(c, 500, "INTERNAL_ERROR", err.Error())
}
