package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/pkg/database"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := newRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status \"ok\", got %v", body["status"])
	}
}

func TestPassportEndpoint_NotFound(t *testing.T) {
	if database.DB == nil {
		t.Skip("skipping: requires a live database connection")
	}

	gin.SetMode(gin.TestMode)

	r := newRouter()

	req := httptest.NewRequest(http.MethodGet, "/p/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if _, ok := body["error"]; !ok {
		t.Errorf("expected an error field in the response body")
	}
}

func TestPassportEndpoint_InvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := newRouter()

	req := httptest.NewRequest(http.MethodGet, "/p/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}
