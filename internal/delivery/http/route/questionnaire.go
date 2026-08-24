package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/questionnaire"
	u "github.com/itauq-golang/internal/usecase/questionnaire"
)

func SetupQuestionnaires(r gin.IRouter, uc *u.Usecase, requireAdmin gin.HandlerFunc) {
	handler := h.NewHandler(uc)
	q := r.Group("/questionnaires", requireAdmin)
	q.POST("", handler.Create)
	q.GET("", handler.List)
	q.GET("/:id", handler.Get)
	q.PATCH("/:id", handler.Update)
	q.DELETE("/:id", handler.Delete)
}
