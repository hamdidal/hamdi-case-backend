package product

import (
	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup) {
	products := rg.Group("/products")
	products.Use(middleware.JWTAuth())
	{
		// Read access: admin and auditor roles may browse the product catalogue.
		products.GET("", middleware.RequireRole("admin", "auditor"), ListProducts)
		products.GET("/:id", middleware.RequireRole("admin", "auditor"), GetProduct)
		products.GET("/:id/qrcode", middleware.RequireRole("admin", "auditor"), GetQRCode)
		products.GET("/:id/pdf", middleware.RequireRole("admin", "auditor"), GetProductPDF)
		products.GET("/:id/versions", middleware.RequireRole("admin", "auditor"), ListVersions)
		products.GET("/:id/versions/:version_number", middleware.RequireRole("admin", "auditor"), GetVersion)

		// Write access: restricted to admin only.
		products.POST("", middleware.RequireRole("admin"), CreateProduct)
		products.PUT("/:id", middleware.RequireRole("admin"), UpdateProduct)
		products.DELETE("/:id", middleware.RequireRole("admin"), DeleteProduct)
	}
}
