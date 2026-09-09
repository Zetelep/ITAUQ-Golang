package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/evaluation"
	u "github.com/itauq-golang/internal/usecase/evaluation"
)

// SetupEvaluation registers the Evaluation Results & Reports endpoints
// (API_SPECIFICATION.md §10). All endpoints require an authenticated
// Administrator or Super Admin; the usecase performs the owner check and
// scopes the list endpoint by role.
func SetupEvaluation(r gin.IRouter, uc *u.Usecase, requireAdmin gin.HandlerFunc) {
	handler := h.NewHandler(uc)

	// /respondents: list + detail
	resp := r.Group("/respondents", requireAdmin)
	resp.GET("", handler.ListRespondents)
	resp.GET("/:id", handler.GetRespondent)

	// /questionnaires/:id/report lives alongside the questionnaire group.
	// Same admin auth gate applies.
	qn := r.Group("/questionnaires", requireAdmin)
	qn.GET("/:id/report", handler.GetQuestionnaireReport)
}
