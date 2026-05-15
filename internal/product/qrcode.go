package product

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

func GetQRCode(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var p models.Product
	err = database.DB.First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve product"})
		return
	}

	baseURL := os.Getenv("PUBLIC_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	passportURL := fmt.Sprintf("%s/p/%s", baseURL, p.ID)

	png, err := qrcode.Encode(passportURL, qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate QR code"})
		return
	}

	c.Data(http.StatusOK, "image/png", png)
}
