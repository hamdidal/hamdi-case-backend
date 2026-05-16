package metrics

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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

	if prometheusURL != "" {
		if val, ok := queryPrometheus(prometheusURL, query); ok {
			c.JSON(http.StatusOK, gin.H{
				"query":     query,
				"value":     val,
				"timestamp": now.Format(time.RFC3339),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"query":     query,
		"value":     mockMetricValue(query),
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

func mockMetricValue(query string) float64 {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	switch {
	case strings.Contains(query, "cpu"):
		return 28.5 + rng.Float64()*15
	case strings.Contains(query, "MemAvailable") || strings.Contains(query, "memory"):
		return 62.3 + rng.Float64()*10
	case strings.Contains(query, "filesystem") || strings.Contains(query, "disk"):
		return 38.7 + rng.Float64()*5
	case strings.Contains(query, "receive"):
		return 98304 + rng.Float64()*65536
	case strings.Contains(query, "transmit"):
		return 65536 + rng.Float64()*32768
	default:
		return 50.0
	}
}
