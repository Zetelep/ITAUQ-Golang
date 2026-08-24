package instrument

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"
)

type Handler struct {
	itauq *itauq.Loaded
	sus   *sus.Loaded
}

func NewHandler(itauqInst *itauq.Loaded, susInst *sus.Loaded) *Handler {
	return &Handler{itauq: itauqInst, sus: susInst}
}

func (h *Handler) GetITAUQ(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.itauq.Instrument})
}

func (h *Handler) GetSUS(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.sus.Instrument})
}
