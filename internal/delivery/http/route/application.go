package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/application"
	u "github.com/itauq-golang/internal/usecase/application"
)

// SetupApplications registers the public submission endpoint and protected review endpoints.
func SetupApplications(r gin.IRouter, uc *u.Usecase, superAdmin gin.HandlerFunc) {
	handler := h.NewHandler(uc)
	r.POST("/applications", handler.Create)
	admin := r.Group("/applications", superAdmin)
	admin.GET("", handler.List)
	admin.GET("/:id", handler.Get)
	admin.PATCH("/:id/approve", handler.Approve)
	admin.PATCH("/:id/reject", handler.Reject)
}
