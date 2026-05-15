package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/auth"
	"github.com/hamdidal/dpp-backend/internal/metrics"
	"github.com/hamdidal/dpp-backend/internal/product"
	"github.com/hamdidal/dpp-backend/internal/user"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"github.com/joho/godotenv"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	database.Connect()

	r := gin.Default()
	r.Use(corsMiddleware())

	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
	})

	api := r.Group("/api/v1")
	auth.RegisterRoutes(api.Group("/auth"))
	product.RegisterRoutes(api)
	user.RegisterRoutes(api)
	metrics.RegisterRoutes(api)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
