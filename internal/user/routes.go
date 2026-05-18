package user

import (
	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")

	// Self-service profile endpoints — any authenticated user.
	me := users.Group("/me")
	me.Use(middleware.JWTAuth())
	{
		me.PUT("", UpdateProfile)
		me.PUT("/password", ChangePassword)
	}

	// Admin-only user-management endpoints.
	admin := users.Group("")
	admin.Use(middleware.JWTAuth(), middleware.RequireRole("admin"))
	{
		admin.GET("", ListUsers)
		admin.PATCH("/:id/role", ChangeRole)
		admin.DELETE("/:id", DeleteUser)
	}
}
