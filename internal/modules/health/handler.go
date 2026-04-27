package health

import (
	"github.com/gin-gonic/gin"
	api "github.com/raufendro/novacore/internal/http"
)

func RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/health", func(c *gin.Context) {
		api.OK(c, "Service is healthy", gin.H{"status": "ok"}, nil)
	})
}
