package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/raufendro/novacore/pkg/security"
	"gorm.io/gorm"
)

func RegisterGenerated(router *gin.RouterGroup, db *gorm.DB, jwt *security.JWTManager) {
	_ = router
	_ = db
	_ = jwt
}
