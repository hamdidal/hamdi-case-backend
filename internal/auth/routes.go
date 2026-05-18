package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/ratelimit"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/login", ratelimit.Login(), Login)
	rg.POST("/register", ratelimit.Register(), Register)
}
