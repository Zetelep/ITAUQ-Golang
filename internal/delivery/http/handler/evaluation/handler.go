package evaluation

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	u "github.com/itauq-golang/internal/usecase/evaluation"
)

type Handler struct{ uc *u.Usecase }

func NewHandler(uc *u.Usecase) *Handler { return &Handler{uc: uc} }

func ok(c *gin.Context, data any, meta any) {
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

func parseInt(v string, d int) int {
	if n, e := strconv.Atoi(v); e == nil && n > 0 {
		return n
	}
	return d
}

// ListRespondents handles GET /respondents.
func (h *Handler) ListRespondents(c *gin.Context) {
	items, meta, err := h.uc.ListRespondents(
		c.Request.Context(),
		getCallerID(c),
		getCallerRole(c),
		c.Query("administrator_id"),
		c.Query("questionnaire_id"),
		parseInt(c.Query("page"), 1),
		parseInt(c.Query("page_size"), 20),
	)
	if err != nil {
		if errors.Is(err, u.ErrForbidden) {
			fail(c, http.StatusForbidden, "FORBIDDEN", err.Error())
			return
		}
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	ok(c, items, meta)
}

// GetRespondent handles GET /respondents/:id.
func (h *Handler) GetRespondent(c *gin.Context) {
	detail, err := h.uc.GetRespondentDetail(
		c.Request.Context(),
		c.Param("id"),
		getCallerID(c),
		getCallerRole(c),
	)
	if err != nil {
		switch {
		case errors.Is(err, u.ErrNotFound):
			fail(c, http.StatusNotFound, "NOT_FOUND", "respondent not found")
		case errors.Is(err, u.ErrForbidden):
			fail(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	ok(c, detail, nil)
}

// GetQuestionnaireReport handles GET /questionnaires/:id/report.
func (h *Handler) GetQuestionnaireReport(c *gin.Context) {
	report, err := h.uc.GetQuestionnaireReport(
		c.Request.Context(),
		c.Param("id"),
		getCallerID(c),
		getCallerRole(c),
	)
	if err != nil {
		switch {
		case errors.Is(err, u.ErrNotFound):
			fail(c, http.StatusNotFound, "NOT_FOUND", "questionnaire not found")
		case errors.Is(err, u.ErrForbidden):
			fail(c, http.StatusForbidden, "FORBIDDEN", err.Error())
		default:
			fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}
	ok(c, report, nil)
}
