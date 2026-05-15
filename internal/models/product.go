package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID               uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	Name             string            `gorm:"not null" json:"name"`
	Description      string            `json:"description"`
	SKU              string            `gorm:"uniqueIndex" json:"sku"`
	Brand            string            `json:"brand"`
	Category         string            `json:"category"`
	Materials        []Material        `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"materials,omitempty"`
	CareInstructions []CareInstruction `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE" json:"care_instructions,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type Material struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID  uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Name       string    `gorm:"not null" json:"name"`
	Percentage float64   `json:"percentage"`
	Origin     string    `json:"origin"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (m *Material) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type CareInstruction struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Type        string    `gorm:"not null" json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ci *CareInstruction) BeforeCreate(tx *gorm.DB) error {
	if ci.ID == uuid.Nil {
		ci.ID = uuid.New()
	}
	return nil
}
