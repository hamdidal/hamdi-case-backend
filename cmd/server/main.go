package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/auth"
	"github.com/hamdidal/dpp-backend/internal/auditlog"
	"github.com/hamdidal/dpp-backend/internal/health"
	"github.com/hamdidal/dpp-backend/internal/metrics"
	"github.com/hamdidal/dpp-backend/internal/product"
	"github.com/hamdidal/dpp-backend/internal/user"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"github.com/joho/godotenv"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using environment variables")
	}

	database.Connect()

	r := newRouter()

	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		slog.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutdown signal received, draining connections...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped cleanly")
}

func newRouter() *gin.Engine {
	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	if allowedOriginsStr == "" {
		allowedOriginsStr = "http://localhost:3001"
	}
	allowedOrigins := strings.Split(allowedOriginsStr, ",")

	r := gin.New()
	r.Use(
		gin.Recovery(),
		requestIDMiddleware(),
		structuredLogger(),
		cors.New(cors.Config{
			AllowOrigins:     allowedOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
	)

	r.GET("/health", health.Handler)

	// Legacy public passport URL (kept for backward compat)
	r.GET("/p/:uuid", product.GetPassport)

	api := r.Group("/api/v1")
	auth.RegisterRoutes(api.Group("/auth"))
	// Public passport under /api/v1 — no auth required
	api.GET("/p/:uuid", product.GetPassport)
	product.RegisterRoutes(api)
	product.RegisterDashboardRoutes(api)
	user.RegisterRoutes(api)
	metrics.RegisterRoutes(api)
	auditlog.RegisterRoutes(api)

	return r
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func structuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		requestID, _ := c.Get("request_id")
		slog.Info("request",
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
