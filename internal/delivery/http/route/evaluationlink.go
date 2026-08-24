package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/evaluationlink"
	u "github.com/itauq-golang/internal/usecase/evaluationlink"
)

func SetupEvaluationLinks(r gin.IRouter, uc *u.Usecase, requireAdmin gin.HandlerFunc) {
	handler := h.NewHandler(uc)

	// Nested under questionnaires: /questionnaires/:id/evaluation-links
	q := r.Group("/questionnaires/:id/evaluation-links", requireAdmin)
	q.GET("", handler.List)
	q.POST("", handler.Create)

	// Top-level for PATCH/DELETE by evaluation link ID
	l := r.Group("/evaluation-links", requireAdmin)
	l.PATCH("/:id", handler.Update)
	l.DELETE("/:id", handler.Delete)
}
