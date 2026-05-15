package product

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/auditlog"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"gorm.io/gorm"
)

// productDiff holds the subset of product fields captured in audit log entries.
type productDiff struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SKU         string `json:"sku"`
	Brand       string `json:"brand"`
	Category    string `json:"category"`
}

func snapshotProduct(p models.Product) productDiff {
	return productDiff{
		Name:        p.Name,
		Description: p.Description,
		SKU:         p.SKU,
		Brand:       p.Brand,
		Category:    p.Category,
	}
}

type productResponse struct {
	models.Product
	QRCodeURL string `json:"qr_code_url"`
}

type productRequest struct {
	Name             string          `json:"name" binding:"required"`
	Description      string          `json:"description"`
	SKU              string          `json:"sku"`
	Brand            string          `json:"brand"`
	Category         string          `json:"category"`
	Materials        []materialInput `json:"materials"`
	CareInstructions []careInstInput `json:"care_instructions"`
}

type materialInput struct {
	Name       string  `json:"name" binding:"required"`
	Percentage float64 `json:"percentage"`
	Origin     string  `json:"origin"`
}

type careInstInput struct {
	Type        string `json:"type" binding:"required"`
	Description string `json:"description"`
}

func ListProducts(c *gin.Context) {
	var products []models.Product
	if err := database.DB.Preload("Materials").Preload("CareInstructions").Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, products)
}

func GetProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var p models.Product
	if err := database.DB.Preload("Materials").Preload("CareInstructions").First(&p, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, productResponse{
		Product:   p,
		QRCodeURL: fmt.Sprintf("/api/v1/products/%s/qrcode", p.ID),
	})
}

func CreateProduct(c *gin.Context) {
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p := models.Product{
		Name:        req.Name,
		Description: req.Description,
		SKU:         req.SKU,
		Brand:       req.Brand,
		Category:    req.Category,
	}
	for _, m := range req.Materials {
		p.Materials = append(p.Materials, models.Material{Name: m.Name, Percentage: m.Percentage, Origin: m.Origin})
	}
	for _, ci := range req.CareInstructions {
		p.CareInstructions = append(p.CareInstructions, models.CareInstruction{Type: ci.Type, Description: ci.Description})
	}

	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	auditlog.LogAction(userID, c.GetString("username"), "create", "product", p.ID, p.Name,
		map[string]any{"after": snapshotProduct(p)})

	c.JSON(http.StatusCreated, p)
}

func UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	// Preload associations so the version snapshot captures the full product state.
	var p models.Product
	if err := database.DB.Preload("Materials").Preload("CareInstructions").First(&p, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	oldSnapshot := snapshotProduct(p)

	// Marshal the full product state before field mutations for the version record.
	snapshotJSON, _ := json.Marshal(p)

	userID, _ := uuid.Parse(c.GetString("user_id"))
	username := c.GetString("username")

	p.Name = req.Name
	p.Description = req.Description
	p.SKU = req.SKU
	p.Brand = req.Brand
	p.Category = req.Category

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Save a version snapshot of the pre-update state inside the transaction
		// so the version and the update are committed atomically.
		var maxVer int
		tx.Model(&models.ProductVersion{}).
			Where("product_id = ?", id).
			Select("COALESCE(MAX(version_number), 0)").
			Scan(&maxVer)

		version := models.ProductVersion{
			ProductID:         id,
			VersionNumber:     maxVer + 1,
			Snapshot:          models.JSONB(snapshotJSON),
			CreatedByID:       userID,
			CreatedByUsername: username,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}

		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&models.Material{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&models.CareInstruction{}).Error; err != nil {
			return err
		}
		for _, m := range req.Materials {
			mat := models.Material{ProductID: id, Name: m.Name, Percentage: m.Percentage, Origin: m.Origin}
			if err := tx.Create(&mat).Error; err != nil {
				return err
			}
		}
		for _, ci := range req.CareInstructions {
			inst := models.CareInstruction{ProductID: id, Type: ci.Type, Description: ci.Description}
			if err := tx.Create(&inst).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	auditlog.LogAction(userID, username, "update", "product", id, req.Name,
		map[string]any{"before": oldSnapshot, "after": snapshotProduct(p)})

	database.DB.Preload("Materials").Preload("CareInstructions").First(&p, "id = ?", id)
	c.JSON(http.StatusOK, p)
}

func DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var p models.Product
	if err := database.DB.First(&p, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	if err := database.DB.Delete(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	auditlog.LogAction(userID, c.GetString("username"), "delete", "product", p.ID, p.Name,
		map[string]any{"before": snapshotProduct(p)})

	c.JSON(http.StatusNoContent, nil)
}
