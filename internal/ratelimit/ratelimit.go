package ratelimit

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// Login allows 5 requests per minute per IP
func Login() gin.HandlerFunc {
	rate := limiter.Rate{Period: 1 * time.Minute, Limit: 5}
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	return mgin.NewMiddleware(instance)
}

// Register allows 3 requests per minute per IP
func Register() gin.HandlerFunc {
	rate := limiter.Rate{Period: 1 * time.Minute, Limit: 3}
	store := memory.NewStore()
	instance := limiter.New(store, rate)
	return mgin.NewMiddleware(instance)
}
