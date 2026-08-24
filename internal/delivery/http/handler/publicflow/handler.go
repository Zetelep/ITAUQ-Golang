package publicflow

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	domain "github.com/itauq-golang/internal/domain/respondent"
	respRepo "github.com/itauq-golang/internal/repository/respondent"
	"github.com/itauq-golang/internal/usecase/publicflow"
)

// startBody mirrors domain.StartInput locally so we don't depend on a single
// JSON-binding struct that also lives in the usecase package.
type startBody = domain.StartInput
type attemptsBody = domain.AttemptsInput
type answersBody = domain.AnswersInput
type susAnswersBody = domain.SUSAnswersInput

type Handler struct{ uc *publicflow.Usecase }

func NewHandler(uc *publicflow.Usecase) *Handler { return &Handler{uc: uc} }

func ok(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"success": true, "data": data})
}

func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"success": false, "error": gin.H{"code": code, "message": message}})
}

func (h *Handler) GetEvaluation(c *gin.Context) {
	payload, err := h.uc.GetEvaluation(c.Request.Context(), c.Param("token"))
	if err != nil {
		mapError(c, err)
		return
	}
	ok(c, http.StatusOK, payload)
}

func (h *Handler) StartRespondent(c *gin.Context) {
	var in startBody
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	started, err := h.uc.StartRespondent(c.Request.Context(), c.Param("token"), in)
	if err != nil {
		mapError(c, err)
		return
	}
	ok(c, http.StatusCreated, started)
}

func (h *Handler) SubmitTaskAttempts(c *gin.Context) {
	var in attemptsBody
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	saved, err := h.uc.SubmitTaskAttempts(c.Request.Context(), c.Param("token"), c.Param("respondent_id"), in)
	if err != nil {
		mapError(c, err)
		return
	}
	ok(c, http.StatusCreated, gin.H{"attempts": saved})
}

func (h *Handler) SubmitAnswers(c *gin.Context) {
	var in answersBody
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	saved, err := h.uc.SubmitAnswers(c.Request.Context(), c.Param("token"), c.Param("respondent_id"), in)
	if err != nil {
		mapError(c, err)
		return
	}
	ok(c, http.StatusCreated, gin.H{"answers": saved})
}

func (h *Handler) SubmitSUSAnswers(c *gin.Context) {
	var in susAnswersBody
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	saved, err := h.uc.SubmitSUSAnswers(c.Request.Context(), c.Param("token"), c.Param("respondent_id"), in)
	if err != nil {
		mapError(c, err)
		return
	}
	ok(c, http.StatusCreated, gin.H{"answers": saved})
}

func (h *Handler) FinalizeSubmit(c *gin.Context) {
	submitted, err := h.uc.FinalizeSubmit(c.Request.Context(), c.Param("token"), c.Param("respondent_id"))
	if err != nil {
		mapError(c, err)
		return
	}
	ok(c, http.StatusOK, submitted)
}

// mapError translates usecase sentinels into HTTP status codes per the API spec.
func mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, publicflow.ErrLinkNotFound):
		fail(c, http.StatusNotFound, "NOT_FOUND", "evaluation link not found")
	case errors.Is(err, publicflow.ErrLinkInactive):
		fail(c, http.StatusConflict, "LINK_INACTIVE", "evaluation link is inactive or expired")
	case errors.Is(err, publicflow.ErrRespondentNotFound):
		fail(c, http.StatusNotFound, "NOT_FOUND", "respondent not found for this evaluation link")
	case errors.Is(err, publicflow.ErrInvalidAnswers):
		fail(c, http.StatusUnprocessableEntity, "INVALID_ANSWERS", err.Error())
	case errors.Is(err, publicflow.ErrInvalidSUSAnswers):
		fail(c, http.StatusUnprocessableEntity, "INVALID_SUS_ANSWERS", err.Error())
	case errors.Is(err, publicflow.ErrInvalidAttempts):
		fail(c, http.StatusUnprocessableEntity, "INVALID_ATTEMPTS", err.Error())
	case errors.Is(err, publicflow.ErrIncomplete):
		fail(c, http.StatusBadRequest, "INCOMPLETE_SESSION", err.Error())
	case errors.Is(err, respRepo.ErrNotFound):
		fail(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
	default:
		// Treat string-level validation errors as 400.
		msg := err.Error()
		if isValidationMessage(msg) {
			fail(c, http.StatusBadRequest, "VALIDATION_ERROR", msg)
			return
		}
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", msg)
	}
}

func isValidationMessage(msg string) bool {
	for _, prefix := range []string{"name is required", "age must be", "gender must be", "duration_seconds must be", "answers must not be empty", "attempts must not be empty", "sus answers must not be empty"} {
		if len(msg) >= len(prefix) && msg[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}