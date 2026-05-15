package auditlog

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
)

// LogAction writes an audit entry after a successful mutation. Errors are
// logged but never propagate to the caller — audit failures must not roll back
// business operations.
func LogAction(userID uuid.UUID, username, action, entityType string, entityID uuid.UUID, entityName string, changes any) {
	raw, err := json.Marshal(changes)
	if err != nil {
		slog.Error("auditlog: failed to marshal changes", "error", err)
		return
	}

	entry := models.AuditLog{
		UserID:     userID,
		Username:   username,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		EntityName: entityName,
		Changes:    models.JSONB(raw),
	}

	if err := database.DB.Create(&entry).Error; err != nil {
		slog.Error("auditlog: failed to write entry", "action", action, "entity_id", entityID, "error", err)
	}
}
