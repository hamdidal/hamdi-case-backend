package product

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"gorm.io/gorm"
)

func ListVersions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var versions []models.ProductVersion
	if err := database.DB.
		Where("product_id = ?", id).
		Order("version_number DESC").
		Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve versions"})
		return
	}

	c.JSON(http.StatusOK, versions)
}

func GetVersion(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	versionNum, err := strconv.Atoi(c.Param("version_number"))
	if err != nil || versionNum < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}

	var version models.ProductVersion
	err = database.DB.
		Where("product_id = ? AND version_number = ?", id, versionNum).
		First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve version"})
		return
	}

	c.JSON(http.StatusOK, version)
}
