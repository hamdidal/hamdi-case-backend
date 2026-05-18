package metrics

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

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

// QueryMetrics handles GET /metrics/query?query=<promql>
// It proxies to Prometheus's instant-query API and returns { query, value, timestamp }.
// Falls back to plausible mock values when Prometheus is unreachable.
func QueryMetrics(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter required"})
		return
	}

	now := time.Now().UTC()
	prometheusURL := os.Getenv("PROMETHEUS_URL")

	if prometheusURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "prometheus not configured"})
		return
	}

	val, ok := queryPrometheus(prometheusURL, query)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "prometheus query returned no data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":     query,
		"value":     val,
		"timestamp": now.Format(time.RFC3339),
	})
}

func queryPrometheus(baseURL, query string) (float64, bool) {
	apiURL := baseURL + "/api/v1/query?query=" + url.QueryEscape(query)
	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Value [2]interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, false
	}
	if result.Status != "success" || len(result.Data.Result) == 0 {
		return 0, false
	}
	valStr, _ := result.Data.Result[0].Value[1].(string)
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

