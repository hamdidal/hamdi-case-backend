package auditlog

import (
	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	// Audit log access is restricted to the admin role.
	logs := rg.Group("/audit-logs")
	logs.Use(middleware.JWTAuth(), middleware.RequireRole("admin"))
	{
		logs.GET("", ListAuditLogs)
	}
}
