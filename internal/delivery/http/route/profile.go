package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/profile"
	u "github.com/itauq-golang/internal/usecase/profile"
)

// SetupProfile registers self-service profile endpoints. The caller must be
// authenticated (a JWT in the Authorization header); RequireAdmin is enough
// because both Administrator and Super Admin use the same /profiles/me shape.
func SetupProfile(r gin.IRouter, uc *u.Usecase, requireAuth gin.HandlerFunc) {
	handler := h.NewHandler(uc)
	me := r.Group("/profiles", requireAuth)
	me.GET("/me", handler.GetMe)
	me.PATCH("/me", handler.UpdateMe)

	// Change-password lives under /auth so it's discoverable alongside login.
	// It accepts any authenticated caller (Administrator or Super Admin); the
	// frontend must call this before navigating away when must_change_password
	// is true on /profiles/me.
	auth := r.Group("/auth", requireAuth)
	auth.POST("/change-password", handler.ChangePassword)
}
