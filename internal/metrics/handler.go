package metrics

import (
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func ProxyMetrics(c *gin.Context) {
	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "prometheus not configured"})
		return
	}

	resp, err := http.Get(prometheusURL + "/metrics")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach prometheus"})
		return
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Header(k, v)
		}
	}
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}
