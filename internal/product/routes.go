package product

import (
	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	products := rg.Group("/products")
	products.Use(middleware.JWTAuth())
	{
		products.GET("", ListProducts)
		products.GET("/:id", GetProduct)
		products.POST("", middleware.RequireRole("admin"), CreateProduct)
		products.PUT("/:id", middleware.RequireRole("admin"), UpdateProduct)
		products.DELETE("/:id", middleware.RequireRole("admin"), DeleteProduct)
	}
}
