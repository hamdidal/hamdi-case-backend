package product

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/internal/testutil"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newProductRouter builds a minimal Gin engine with the product routes wired,
// bypassing JWT auth so tests stay focused on handler behaviour.
// The callerRole is injected directly into the context.
func newProductRouter(callerUserID, callerUsername, callerRole string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	injectClaims := func(c *gin.Context) {
		c.Set("user_id", callerUserID)
		c.Set("username", callerUsername)
		c.Set("role", callerRole)
		c.Next()
	}

	r.GET("/products", injectClaims, ListProducts)
	r.GET("/products/:id", injectClaims, GetProduct)
	r.POST("/products", injectClaims, CreateProduct)
	r.PUT("/products/:id", injectClaims, UpdateProduct)
	r.DELETE("/products/:id", injectClaims, DeleteProduct)
	return r
}

// seedProduct inserts a product record directly through the ORM for test
// setup, bypassing the HTTP handler.
func seedProduct(t *testing.T, name, status string, mats []models.Material) models.Product {
	t.Helper()
	// Use a unique SKU to respect the uniqueIndex constraint.
	p := models.Product{
		Name:     name,
		Brand:    "TestBrand",
		Category: "t-shirt",
		Status:   status,
		SKU:      fmt.Sprintf("TEST-SKU-%s", uuid.New().String()),
	}
	p.Materials = mats
	require.NoError(t, database.DB.Create(&p).Error)
	return p
}

// ─── GetProduct ──────────────────────────────────────────────────────────────

func TestGetProduct(t *testing.T) {
	defer testutil.NewTestDB(t)()
	r := newProductRouter("admin-uuid", "admin", "admin")

	existing := seedProduct(t, "Cotton Hoodie", "published", []models.Material{
		{Name: "Cotton", Percentage: 100},
	})

	tests := []struct {
		name       string
		id         string
		wantStatus int
		wantName   string
	}{
		{
			name:       "existing product returns 200",
			id:         existing.ID.String(),
			wantStatus: http.StatusOK,
			wantName:   "Cotton Hoodie",
		},
		{
			name:       "non-existent UUID returns 404",
			id:         uuid.New().String(),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "malformed UUID returns 400",
			id:         "not-a-valid-uuid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/"+tc.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantName != "" {
				var body map[string]any
				require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
				assert.Equal(t, tc.wantName, body["name"])
			}
		})
	}
}

// ─── ListProducts ─────────────────────────────────────────────────────────────

func TestListProducts_Pagination(t *testing.T) {
	defer testutil.NewTestDB(t)()
	r := newProductRouter("admin-uuid", "admin", "admin")

	// Insert 5 products.
	for i := 0; i < 5; i++ {
		seedProduct(t, fmt.Sprintf("Product %d", i), "published", nil)
	}

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantCount  int
	}{
		{
			name:       "default page returns all 5 products",
			query:      "",
			wantStatus: http.StatusOK,
			wantCount:  5,
		},
		{
			name:       "limit=2 returns 2 products",
			query:      "?page=1&limit=2",
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
		{
			name:       "page=3 limit=2 returns 1 product",
			query:      "?page=3&limit=2",
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "limit > 100 clamps to 20",
			query:      "?limit=999",
			wantStatus: http.StatusOK,
			wantCount:  5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products"+tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)
			var body map[string]any
			require.NoError(t, json.NewDecoder(w.Body).Decode(&body))

			data, ok := body["data"].([]any)
			require.True(t, ok, "response.data must be an array")
			assert.Len(t, data, tc.wantCount)
		})
	}
}

// ─── CreateProduct — material composition validation ─────────────────────────

func TestCreateProduct_MaterialValidation(t *testing.T) {
	defer testutil.NewTestDB(t)()
	r := newProductRouter("admin-uuid", "admin", "admin")

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
	}{
		{
			name: "valid: materials sum to exactly 100%",
			payload: map[string]any{
				"name":   "Valid Product",
				"brand":  "Brand",
				"materials": []map[string]any{
					{"name": "Cotton", "percentage": 60.0, "recycled": false},
					{"name": "Polyester", "percentage": 40.0, "recycled": false},
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "valid: single 100% material",
			payload: map[string]any{
				"name":  "Pure Wool",
				"brand": "Brand",
				"materials": []map[string]any{
					{"name": "Merino Wool", "percentage": 100.0, "recycled": false},
				},
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "valid: no materials (empty slice allowed)",
			payload: map[string]any{
				"name":  "No Materials",
				"brand": "Brand",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "invalid: sum 99.9% rejected with 422",
			payload: map[string]any{
				"name":  "Under Filled",
				"brand": "Brand",
				"materials": []map[string]any{
					{"name": "Cotton", "percentage": 60.0},
					{"name": "Polyester", "percentage": 39.9},
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid: sum 105% rejected with 422",
			payload: map[string]any{
				"name":  "Over Filled",
				"brand": "Brand",
				"materials": []map[string]any{
					{"name": "Cotton", "percentage": 60.0},
					{"name": "Polyester", "percentage": 45.0},
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid: missing required name field",
			payload: map[string]any{
				"brand": "Brand",
				"materials": []map[string]any{
					{"name": "Cotton", "percentage": 100.0},
				},
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

func TestCreateProduct_ErrorBodyContainsMaterialMessage(t *testing.T) {
	defer testutil.NewTestDB(t)()
	r := newProductRouter("admin-uuid", "admin", "admin")

	payload := map[string]any{
		"name": "Bad Product",
		"materials": []map[string]any{
			{"name": "Cotton", "percentage": 55.5},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Contains(t, resp["error"].(string), "100%")
}

// ─── DeleteProduct ────────────────────────────────────────────────────────────

func TestDeleteProduct(t *testing.T) {
	defer testutil.NewTestDB(t)()
	r := newProductRouter("admin-uuid", "admin", "admin")

	existing := seedProduct(t, "To Delete", "draft", nil)

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "delete existing product returns 204",
			id:         existing.ID.String(),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete non-existent product returns 404",
			id:         uuid.New().String(),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "delete with malformed ID returns 400",
			id:         "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/products/"+tc.id, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

// ─── UpdateProduct — material validation ─────────────────────────────────────

func TestUpdateProduct_MaterialValidation(t *testing.T) {
	defer testutil.NewTestDB(t)()
	r := newProductRouter("admin-uuid", "admin", "admin")

	existing := seedProduct(t, "Existing", "published", []models.Material{
		{Name: "Cotton", Percentage: 100},
	})
	id := existing.ID.String()

	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus int
	}{
		{
			name: "update with valid 100% materials succeeds",
			payload: map[string]any{
				"name": "Updated Product",
				"materials": []map[string]any{
					{"name": "Cotton", "percentage": 70.0},
					{"name": "Elastan", "percentage": 30.0},
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "update with 99.9% materials rejected",
			payload: map[string]any{
				"name": "Bad Update",
				"materials": []map[string]any{
					{"name": "Cotton", "percentage": 60.0},
					{"name": "Elastan", "percentage": 39.9},
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPut, "/products/"+id, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}
