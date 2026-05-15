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

type PublicMaterial struct {
	Name       string  `json:"name"`
	Percentage float64 `json:"percentage"`
	Origin     string  `json:"origin,omitempty"`
}

type PublicCareInstruction struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type PublicPassportResponse struct {
	Name             string                  `json:"name"`
	Brand            string                  `json:"brand,omitempty"`
	Category         string                  `json:"category,omitempty"`
	Materials        []PublicMaterial        `json:"materials,omitempty"`
	CareInstructions []PublicCareInstruction `json:"care_instructions,omitempty"`
}

func GetPassport(c *gin.Context) {
	id, err := uuid.Parse(c.Param("uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passport id"})
		return
	}

	var p models.Product
	err = database.DB.
		Preload("Materials").
		Preload("CareInstructions").
		First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "passport not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve passport"})
		return
	}

	c.JSON(http.StatusOK, toPublicPassport(p))
}

func toPublicPassport(p models.Product) PublicPassportResponse {
	materials := make([]PublicMaterial, len(p.Materials))
	for i, m := range p.Materials {
		materials[i] = PublicMaterial{
			Name:       m.Name,
			Percentage: m.Percentage,
			Origin:     m.Origin,
		}
	}

	care := make([]PublicCareInstruction, len(p.CareInstructions))
	for i, ci := range p.CareInstructions {
		care[i] = PublicCareInstruction{
			Type:        ci.Type,
			Description: ci.Description,
		}
	}

	return PublicPassportResponse{
		Name:             p.Name,
		Brand:            p.Brand,
		Category:         p.Category,
		Materials:        materials,
		CareInstructions: care,
	}
}
