package product

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"gorm.io/gorm"
)

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
	c.JSON(http.StatusOK, p)
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
	c.JSON(http.StatusCreated, p)
}

func UpdateProduct(c *gin.Context) {
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

	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p.Name = req.Name
	p.Description = req.Description
	p.SKU = req.SKU
	p.Brand = req.Brand
	p.Category = req.Category

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
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

	database.DB.Preload("Materials").Preload("CareInstructions").First(&p, "id = ?", id)
	c.JSON(http.StatusOK, p)
}

func DeleteProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	if err := database.DB.Delete(&models.Product{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
