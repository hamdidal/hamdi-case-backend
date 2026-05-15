package user

import (
	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	// All user-management operations are restricted to the admin role.
	users := rg.Group("/users")
	users.Use(middleware.JWTAuth(), middleware.RequireRole("admin"))
	{
		users.GET("", ListUsers)
		users.PATCH("/:id/role", ChangeRole)
		users.DELETE("/:id", DeleteUser)
	}
}
