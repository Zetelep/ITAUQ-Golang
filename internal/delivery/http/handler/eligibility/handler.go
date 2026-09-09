package eligibility

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	domain "github.com/itauq-golang/internal/domain/eligibility"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	repo "github.com/itauq-golang/internal/repository/eligibility"
	u "github.com/itauq-golang/internal/usecase/eligibility"
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
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	callerID := getCallerID(c)
	callerRole := getCallerRole(c)
	questionnaireID := c.Param("id")

	crit, err := h.uc.Create(c.Request.Context(), questionnaireID, in, callerID, callerRole)
	if err != nil {
		handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": crit})
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

func (h *Handler) Get(c *gin.Context) {
	callerID := getCallerID(c)
	callerRole := getCallerRole(c)

	crit, err := h.uc.Get(c.Request.Context(), c.Param("id"), callerID, callerRole)
	if err != nil {
		handleError(c, err)
		return
	}
	ok(c, crit)
}

func (h *Handler) Update(c *gin.Context) {
	var in domain.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	callerID := getCallerID(c)

	crit, err := h.uc.Update(c.Request.Context(), c.Param("id"), in, callerID)
	if err != nil {
		handleError(c, err)
		return
	}
	ok(c, crit)
}

func (h *Handler) Delete(c *gin.Context) {
	callerID := getCallerID(c)

	err := h.uc.Delete(c.Request.Context(), c.Param("id"), callerID)
	if err != nil {
		handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func handleError(c *gin.Context, err error) {
	if errors.Is(err, repo.ErrNotFound) || errors.Is(err, qRepo.ErrNotFound) {
		fail(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, qRepo.ErrForbidden) {
		fail(c, http.StatusForbidden, "FORBIDDEN", "not authorized to access this resource")
		return
	}
	fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
}
