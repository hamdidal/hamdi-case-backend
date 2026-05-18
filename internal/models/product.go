package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey"                                      json:"id"`
	UUID           string       `gorm:"-"                                                          json:"uuid"`
	Name           string       `gorm:"not null;index"                                             json:"name"`
	Brand          string       `gorm:"index"                                                      json:"brand"`
	Category       string       `gorm:"index"                                                      json:"category"`
	Country        string       `                                                                  json:"country"`
	ProductionDate string       `                                                                  json:"productionDate"`
	Status         string       `gorm:"default:'draft';index"                                      json:"status"`
	CreatedBy      string       `                                                                  json:"createdBy,omitempty"`
	Materials      []Material   `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"           json:"materials"`
	Care           *ProductCare `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"           json:"careInstructions,omitempty"`
	CreatedAt      time.Time    `gorm:"index"                                                      json:"createdAt"`
	UpdatedAt      time.Time    `                                                                  json:"updatedAt"`
	// Legacy fields kept for backward compat — not used by the main API.
	Description string `json:"description,omitempty"`
	SKU         string `gorm:"uniqueIndex" json:"sku,omitempty"`
}

func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// AfterFind populates the virtual UUID field from the primary key.
func (p *Product) AfterFind(_ *gorm.DB) error {
	p.UUID = p.ID.String()
	return nil
}

// AfterCreate populates UUID after insert.
func (p *Product) AfterCreate(_ *gorm.DB) error {
	p.UUID = p.ID.String()
	return nil
}

// Material is a single fibre/substance component of a product.
type Material struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	ProductID  uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`
	Name       string    `gorm:"not null"             json:"name"`
	Percentage float64   `                            json:"percentage"`
	Recycled   bool      `                            json:"recycled"`
	Origin     string    `                            json:"origin,omitempty"`
	CreatedAt  time.Time `                            json:"-"`
	UpdatedAt  time.Time `                            json:"-"`
}

func (m *Material) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// ProductCare holds the structured care-instruction record for a product (1:1).
type ProductCare struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"             json:"-"`
	ProductID       uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"   json:"-"`
	WashTemperature string    `                                         json:"washTemperature,omitempty"`
	Ironing         string    `                                         json:"ironing,omitempty"`
	DryClean        bool      `                                         json:"dryClean"`
	Bleaching       bool      `                                         json:"bleaching"`
	Notes           string    `                                         json:"notes,omitempty"`
	CreatedAt       time.Time `                                         json:"-"`
	UpdatedAt       time.Time `                                         json:"-"`
}

func (pc *ProductCare) BeforeCreate(tx *gorm.DB) error {
	if pc.ID == uuid.Nil {
		pc.ID = uuid.New()
	}
	return nil
}

// CareInstruction is the legacy per-row model kept in the DB but no longer
// surfaced by the main product API.  Do not remove — AutoMigrate would attempt
// a schema change if the struct disappears before the table is manually dropped.
type CareInstruction struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null"   json:"product_id"`
	Type        string    `gorm:"not null"             json:"type"`
	Description string    `                            json:"description"`
	CreatedAt   time.Time `                            json:"created_at"`
	UpdatedAt   time.Time `                            json:"updated_at"`
}

func (ci *CareInstruction) BeforeCreate(tx *gorm.DB) error {
	if ci.ID == uuid.Nil {
		ci.ID = uuid.New()
	}
	return nil
}
