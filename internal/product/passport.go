package product

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"gorm.io/gorm"
)

// GetPassport handles the public (no-auth) product passport endpoint.
// It is registered at both /p/:uuid (legacy) and /api/v1/p/:uuid (current).
func GetPassport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passport id"})
		return
	}

	var p models.Product
	err = database.DB.
		Preload("Materials").
		Preload("Care").
		First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "passport not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve passport"})
		return
	}

	// Return the full product structure (same as the authenticated endpoint)
	// so the public page can render all fields.
	c.JSON(http.StatusOK, p)
}
