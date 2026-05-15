package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/metrics", middleware.JWTAuth(), middleware.RequireRole("admin"), ProxyMetrics)
}
