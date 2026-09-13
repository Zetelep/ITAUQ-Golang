package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/taskscenario"
	u "github.com/itauq-golang/internal/usecase/taskscenario"
)

func SetupTaskScenarios(r gin.IRouter, uc *u.Usecase, requireAdmin gin.HandlerFunc) {
	handler := h.NewHandler(uc)

	// Nested under questionnaires: /questionnaires/:id/task-scenarios
	q := r.Group("/questionnaires/:id/task-scenarios", requireAdmin)
	q.GET("", handler.List)
	q.GET("/stats", handler.GetStats)
	q.POST("", handler.Create)

	// Top-level for PATCH/DELETE by task scenario ID
	ts := r.Group("/task-scenarios", requireAdmin)
	ts.GET("/:id", handler.Get)
	ts.PATCH("/:id", handler.Update)
	ts.DELETE("/:id", handler.Delete)
}
