package product

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/auditlog"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"gorm.io/gorm"
)

type matIn struct {
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
	Recycled   bool    `json:"recycled"`
}

type careIn struct {
	WashTemperature string `json:"washTemperature"`
	Ironing         string `json:"ironing"`
	DryClean        bool   `json:"dryClean"`
	Bleaching       bool   `json:"bleaching"`
	Notes           string `json:"notes"`
}

type productRequest struct {
	Name             string   `json:"name" binding:"required"`
	Brand            string   `json:"brand"`
	Category         string   `json:"category"`
	Country          string   `json:"country"`
	ProductionDate   string   `json:"productionDate"`
	Status           string   `json:"status"`
	Materials        []matIn  `json:"materials"`
	CareInstructions *careIn  `json:"careInstructions"`
}

type productSnap struct {
	Name           string `json:"name"`
	Brand          string `json:"brand"`
	Category       string `json:"category"`
	Country        string `json:"country"`
	ProductionDate string `json:"productionDate"`
	Status         string `json:"status"`
}

func snap(p models.Product) productSnap {
	return productSnap{
		Name:           p.Name,
		Brand:          p.Brand,
		Category:       p.Category,
		Country:        p.Country,
		ProductionDate: p.ProductionDate,
		Status:         p.Status,
	}
}

// validateMaterials returns an error when the sum of all material percentages
// falls outside the [99.99, 100.01] range (±0.01 floating-point tolerance).
// An empty slice is considered valid — the caller decides whether materials
// are required.
func validateMaterials(mats []matIn) error {
	if len(mats) == 0 {
		return nil
	}
	var total float64
	for _, m := range mats {
		total += m.Percentage
	}
	if total < 99.99 || total > 100.01 {
		return fmt.Errorf("material percentages must sum to exactly 100%% (got %.2f%%)", total)
	}
	return nil
}

func GetDashboardStats(c *gin.Context) {
	type matStat struct {
		Name       string  `json:"name"`
		Percentage float64 `json:"percentage"`
	}
	type productStat struct {
		Category  string    `json:"category"`
		Brand     string    `json:"brand"`
		CreatedAt string    `json:"createdAt"`
		Materials []matStat `json:"materials"`
	}

	var products []models.Product
	if err := database.DB.
		Select("id, category, brand, created_at").
		Preload("Materials", func(db *gorm.DB) *gorm.DB {
			return db.Select("product_id, name, percentage")
		}).
		Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	stats := make([]productStat, len(products))
	for i, p := range products {
		mats := make([]matStat, len(p.Materials))
		for j, m := range p.Materials {
			mats[j] = matStat{Name: m.Name, Percentage: m.Percentage}
		}
		stats[i] = productStat{
			Category:  p.Category,
			Brand:     p.Brand,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Materials: mats,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func ListProducts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	category := c.Query("category")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := database.DB.Model(&models.Product{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("name ILIKE ? OR brand ILIKE ?", like, like)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var products []models.Product
	if err := q.Preload("Materials").Preload("Care").
		Order("created_at DESC").Limit(limit).Offset(offset).
		Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  products,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func GetProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var p models.Product
	if err := database.DB.Preload("Materials").Preload("Care").
		First(&p, "id = ?", id).Error; err != nil {
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

	if err := validateMaterials(req.Materials); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	status := req.Status
	if status == "" {
		status = "draft"
	}

	p := models.Product{
		Name:           req.Name,
		Brand:          req.Brand,
		Category:       req.Category,
		Country:        req.Country,
		ProductionDate: req.ProductionDate,
		Status:         status,
		CreatedBy:      c.GetString("username"),
		// Auto-generate SKU so the unique index is never violated by API-created products.
		SKU: uuid.New().String(),
	}
	for _, m := range req.Materials {
		p.Materials = append(p.Materials, models.Material{
			Name:       m.Name,
			Percentage: m.Percentage,
			Recycled:   m.Recycled,
		})
	}
	if req.CareInstructions != nil {
		p.Care = &models.ProductCare{
			WashTemperature: req.CareInstructions.WashTemperature,
			Ironing:         req.CareInstructions.Ironing,
			DryClean:        req.CareInstructions.DryClean,
			Bleaching:       req.CareInstructions.Bleaching,
			Notes:           req.CareInstructions.Notes,
		}
	}

	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	auditlog.LogAction(userID, c.GetString("username"), "create", "product", p.ID, p.Name,
		map[string]any{"after": snap(p)})

	c.JSON(http.StatusCreated, p)
}

func UpdateProduct(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var p models.Product
	if err := database.DB.Preload("Materials").Preload("Care").
		First(&p, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateMaterials(req.Materials); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	oldFull := fullSnapFromProduct(p)
	snapshotJSON, _ := json.Marshal(p)

	userID, _ := uuid.Parse(c.GetString("user_id"))
	username := c.GetString("username")

	p.Name = req.Name
	p.Brand = req.Brand
	p.Category = req.Category
	p.Country = req.Country
	p.ProductionDate = req.ProductionDate
	if req.Status != "" {
		p.Status = req.Status
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Save version snapshot
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

		// Replace materials
		if err := tx.Where("product_id = ?", id).Delete(&models.Material{}).Error; err != nil {
			return err
		}
		for _, m := range req.Materials {
			mat := models.Material{
				ProductID:  id,
				Name:       m.Name,
				Percentage: m.Percentage,
				Recycled:   m.Recycled,
			}
			if err := tx.Create(&mat).Error; err != nil {
				return err
			}
		}

		// Replace care
		tx.Where("product_id = ?", id).Delete(&models.ProductCare{})
		if req.CareInstructions != nil {
			care := models.ProductCare{
				ProductID:       id,
				WashTemperature: req.CareInstructions.WashTemperature,
				Ironing:         req.CareInstructions.Ironing,
				DryClean:        req.CareInstructions.DryClean,
				Bleaching:       req.CareInstructions.Bleaching,
				Notes:           req.CareInstructions.Notes,
			}
			if err := tx.Create(&care).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newFull := fullSnapFromReq(req, p.Status)
	auditlog.LogAction(userID, username, "update", "product", id, req.Name, map[string]any{
		"before": oldFull,
		"after":  newFull,
		"diff":   deepDiff(oldFull, newFull),
	})

	// Return freshly-loaded product
	database.DB.Preload("Materials").Preload("Care").First(&p, "id = ?", id)
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
		map[string]any{"before": snap(p)})

	c.JSON(http.StatusNoContent, nil)
}
