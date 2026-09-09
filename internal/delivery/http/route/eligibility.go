package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/eligibility"
	u "github.com/itauq-golang/internal/usecase/eligibility"
)

// SetupEligibility mounts the admin CRUD for the "Terms & Conditions
// checklist" on each questionnaire. See API_SPECIFICATION.md §6b.
func SetupEligibility(r gin.IRouter, uc *u.Usecase, requireAdmin gin.HandlerFunc) {
	handler := h.NewHandler(uc)

	// Nested under questionnaires: /questionnaires/:id/eligibility-criteria
	q := r.Group("/questionnaires/:id/eligibility-criteria", requireAdmin)
	q.GET("", handler.List)
	q.POST("", handler.Create)

	// Top-level for GET/PATCH/DELETE by criterion ID
	ec := r.Group("/eligibility-criteria", requireAdmin)
	ec.GET("/:id", handler.Get)
	ec.PATCH("/:id", handler.Update)
	ec.DELETE("/:id", handler.Delete)
}
