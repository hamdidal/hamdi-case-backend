package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	// System metrics are read-only; both admin and auditor roles may query them.
	rg.GET("/metrics", middleware.JWTAuth(), middleware.RequireRole("admin", "auditor"), ProxyMetrics)
}
