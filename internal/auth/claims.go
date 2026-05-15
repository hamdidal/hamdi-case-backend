package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/hamdidal/dpp-backend/internal/models"
)

type Claims struct {
	UserID   string      `json:"user_id"`
	Username string      `json:"username"`
	Role     models.Role `json:"role"`
	jwt.RegisteredClaims
}
