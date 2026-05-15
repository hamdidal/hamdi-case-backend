package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func Handler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
}
