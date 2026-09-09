package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/administrator"
	u "github.com/itauq-golang/internal/usecase/administrator"
)

// SetupAdministrators registers the /administrators endpoints. All routes are
// gated behind superAdmin so only Super Admin can manage administrator
// accounts (per API_SPEC.md §4 and the role permission matrix in §11).
func SetupAdministrators(r gin.IRouter, uc *u.Usecase, superAdmin gin.HandlerFunc) {
	handler := h.NewHandler(uc)
	g := r.Group("/administrators", superAdmin)
	g.GET("", handler.List)
	g.GET("/:id", handler.Get)
	g.POST("", handler.Create)
	g.PATCH("/:id", handler.Update)
	g.DELETE("/:id", handler.Delete)
}
