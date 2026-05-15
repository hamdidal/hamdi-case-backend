package auditlog

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
)

func ListAuditLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	q := database.DB.Model(&models.AuditLog{})

	if v := c.Query("entity_id"); v != "" {
		q = q.Where("entity_id = ?", v)
	}
	if v := c.Query("user_id"); v != "" {
		q = q.Where("user_id = ?", v)
	}
	if v := c.Query("action"); v != "" {
		q = q.Where("action = ?", v)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count audit logs"})
		return
	}

	var logs []models.AuditLog
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"data":   logs,
	})
}
