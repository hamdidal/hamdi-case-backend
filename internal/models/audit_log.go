package models

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSONB stores arbitrary JSON in a PostgreSQL jsonb column.
type JSONB []byte

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *JSONB) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append((*j)[0:0], v...)
	case string:
		*j = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into JSONB", value)
	}
	return nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	*j = append((*j)[0:0], data...)
	return nil
}

type AuditLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Username   string    `gorm:"not null" json:"username"`
	Action     string    `gorm:"not null" json:"action"`
	EntityType string    `gorm:"not null" json:"entity_type"`
	EntityID   uuid.UUID `gorm:"type:uuid;not null;index" json:"entity_id"`
	EntityName string    `json:"entity_name"`
	Changes    JSONB     `gorm:"type:jsonb" json:"changes"`
	CreatedAt  time.Time `json:"created_at"`
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
