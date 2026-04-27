package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/raufendro/novacore/internal/middleware"
	"github.com/raufendro/novacore/pkg/security"
)

func RegisterRoutes(router *gin.RouterGroup, handler *Handler, jwt *security.JWTManager) {
	group := router.Group("/auth")
	group.POST("/register", handler.Register)
	group.POST("/login", handler.Login)
	group.POST("/refresh", handler.Refresh)
	group.POST("/logout", middleware.Auth(jwt), handler.Logout)
	group.GET("/me", middleware.Auth(jwt), handler.Me)
}
