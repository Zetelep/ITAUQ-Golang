package route

import (
	"github.com/gin-gonic/gin"
	h "github.com/itauq-golang/internal/delivery/http/handler/instrument"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"
)

func SetupInstruments(r gin.IRouter, instrument *itauq.Loaded, susInstrument *sus.Loaded, requireAdmin gin.HandlerFunc) {
	handler := h.NewHandler(instrument, susInstrument)
	g := r.Group("/instruments", requireAdmin)
	g.GET("/itauq", handler.GetITAUQ)
	g.GET("/sus", handler.GetSUS)
}
