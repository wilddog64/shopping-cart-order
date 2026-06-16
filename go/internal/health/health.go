package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	pinger  Pinger
	version string
}

func NewHandler(pinger Pinger, version string) *Handler {
	return &Handler{pinger: pinger, version: version}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func (h *Handler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func (h *Handler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if h.pinger != nil {
		if err := h.pinger.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func (h *Handler) Info(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": h.version})
}
