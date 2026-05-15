package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductVersion is an immutable snapshot of a product's full state captured
// immediately before each update is applied. Versions are never modified after
// creation — there is no UpdatedAt field by design.
type ProductVersion struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID         uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	VersionNumber     int       `gorm:"not null" json:"version_number"`
	Snapshot          JSONB     `gorm:"type:jsonb" json:"snapshot"`
	CreatedByID       uuid.UUID `gorm:"type:uuid;not null" json:"created_by_id"`
	CreatedByUsername string    `gorm:"not null" json:"created_by_username"`
	CreatedAt         time.Time `json:"created_at"`
}

func (v *ProductVersion) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
