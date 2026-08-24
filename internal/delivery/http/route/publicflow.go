package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/publicflow"
	u "github.com/itauq-golang/internal/usecase/publicflow"
)

// SetupPublicFlow registers the anonymous respondent endpoints under
// /public/evaluation. These endpoints are mounted *without* auth middleware —
// the route is keyed by the short token in the URL.
func SetupPublicFlow(r gin.IRouter, uc *u.Usecase) {
	handler := h.NewHandler(uc)
	g := r.Group("/public/evaluation")
	g.GET("/:token", handler.GetEvaluation)
	g.POST("/:token/respondents", handler.StartRespondent)
	g.POST("/:token/respondents/:respondent_id/task-attempts", handler.SubmitTaskAttempts)
	g.POST("/:token/respondents/:respondent_id/answers", handler.SubmitAnswers)
	g.POST("/:token/respondents/:respondent_id/sus-answers", handler.SubmitSUSAnswers)
	g.POST("/:token/respondents/:respondent_id/submit", handler.FinalizeSubmit)
}