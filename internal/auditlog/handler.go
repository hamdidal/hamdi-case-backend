package auditlog

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/pkg/database"
)

func ListAuditLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	if page < 1 {
		page = 1
	}

	// When page is provided without offset, derive offset from page
	if c.Query("page") != "" && c.Query("offset") == "" {
		offset = (page - 1) * limit
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
	if v := c.Query("username"); v != "" {
		q = q.Where("username ILIKE ?", "%"+v+"%")
	}

	// Date range filtering — frontend sends ISO-8601 strings as dateFrom / dateTo
	if v := c.Query("dateFrom"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			q = q.Where("created_at >= ?", t)
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if v := c.Query("dateTo"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			// Include the full end day
			q = q.Where("created_at <= ?", t.Add(24*time.Hour))
		} else if t, err := time.Parse("2006-01-02", v); err == nil {
			q = q.Where("created_at <= ?", t.Add(24*time.Hour))
		}
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
		"data":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
