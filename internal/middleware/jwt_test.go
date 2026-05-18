package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hamdidal/dpp-backend/internal/auth"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/stretchr/testify/assert"
)

const testSecret = "middleware-unit-test-secret"

// makeToken creates a signed HS256 JWT using auth.Claims so the middleware can
// parse it correctly with its own claims struct.
func makeToken(t *testing.T, secret, userID, username, role string, ttl time.Duration) string {
	t.Helper()
	claims := auth.Claims{
		UserID:   userID,
		Username: username,
		Role:     models.Role(role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return signed
}

// newMWRouter wraps the middleware under test in a minimal Gin engine whose
// single protected route echoes the context values set by the middleware.
func newMWRouter(mw ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", append(mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":  c.GetString("user_id"),
			"username": c.GetString("username"),
			"role":     c.GetString("role"),
		})
	})...)
	return r
}

// ─── JWTAuth ──────────────────────────────────────────────────────────────────

func TestJWTAuth(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	validToken := makeToken(t, testSecret, "user-uuid-1", "alice", "auditor", time.Hour)
	expiredToken := makeToken(t, testSecret, "user-uuid-2", "bob", "admin", -time.Minute)

	tests := []struct {
		name        string
		authHeader  string
		wantStatus  int
		wantUserID  string
		wantRole    string
	}{
		{
			name:       "valid Bearer token passes through",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
			wantUserID: "user-uuid-1",
			wantRole:   "auditor",
		},
		{
			name:       "missing Authorization header returns 401",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "non-Bearer scheme returns 401",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed token returns 401",
			authHeader: "Bearer not.a.valid.token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "expired token returns 401",
			authHeader: "Bearer " + expiredToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "token signed with wrong secret returns 401",
			authHeader: "Bearer " + makeToken(t, "wrong-secret", "x", "x", "admin", time.Hour),
			wantStatus: http.StatusUnauthorized,
		},
	}

	r := newMWRouter(JWTAuth())

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

func TestJWTAuth_ContextValues(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	token := makeToken(t, testSecret, "uuid-abc-123", "charlie", "admin", time.Hour)
	r := newMWRouter(JWTAuth())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Verify the middleware injects the correct values into the context.
	assert.Contains(t, w.Body.String(), "uuid-abc-123")
	assert.Contains(t, w.Body.String(), "charlie")
	assert.Contains(t, w.Body.String(), "admin")
}

// ─── RequireRole ─────────────────────────────────────────────────────────────

// injectRole is a test middleware that bypasses JWT auth and directly injects
// a role string into the gin context. This isolates RequireRole from JWTAuth.
func injectRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("role", role)
		c.Next()
	}
}

func newRoleRouter(allowedRoles ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin", injectRole("admin"), RequireRole("admin"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.GET("/any", injectRole("auditor"), RequireRole(allowedRoles...), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		injectAsRole string
		allowedRoles []string
		wantStatus   int
	}{
		{
			name:         "admin accessing admin-only route succeeds",
			injectAsRole: "admin",
			allowedRoles: []string{"admin"},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "auditor accessing admin-only route returns 403",
			injectAsRole: "auditor",
			allowedRoles: []string{"admin"},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "auditor accessing multi-role route succeeds",
			injectAsRole: "auditor",
			allowedRoles: []string{"admin", "auditor"},
			wantStatus:   http.StatusOK,
		},
		{
			name:         "unknown role with admin-only route returns 403",
			injectAsRole: "guest",
			allowedRoles: []string{"admin"},
			wantStatus:   http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.GET("/route", injectRole(tc.injectAsRole), RequireRole(tc.allowedRoles...), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/route", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

// TestRBAC_ProductEndpoints asserts that the auditor role is rejected on
// write-only product endpoints (POST, PUT, DELETE) which are guarded by
// RequireRole("admin").
func TestRBAC_ProductEndpoints(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	auditorToken := makeToken(t, testSecret, "auditor-uuid", "auditor1", "auditor", time.Hour)
	adminToken := makeToken(t, testSecret, "admin-uuid", "admin1", "admin", time.Hour)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Simulate the real route structure: JWT auth followed by role guard.
	adminOnly := r.Group("/api/v1/products")
	adminOnly.Use(JWTAuth(), RequireRole("admin"))
	adminOnly.POST("", func(c *gin.Context) { c.Status(http.StatusCreated) })
	adminOnly.DELETE("/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	readOnly := r.Group("/api/v1/products")
	readOnly.Use(JWTAuth(), RequireRole("admin", "auditor"))
	readOnly.GET("", func(c *gin.Context) { c.Status(http.StatusOK) })

	scenarios := []struct {
		name       string
		method     string
		path       string
		token      string
		wantStatus int
	}{
		{"auditor POST products → 403", http.MethodPost, "/api/v1/products", auditorToken, http.StatusForbidden},
		{"auditor DELETE product → 403", http.MethodDelete, "/api/v1/products/some-uuid", auditorToken, http.StatusForbidden},
		{"auditor GET products → 200", http.MethodGet, "/api/v1/products", auditorToken, http.StatusOK},
		{"admin POST products → 201", http.MethodPost, "/api/v1/products", adminToken, http.StatusCreated},
		{"admin DELETE product → 204", http.MethodDelete, "/api/v1/products/some-uuid", adminToken, http.StatusNoContent},
		{"unauthenticated GET products → 401", http.MethodGet, "/api/v1/products", "", http.StatusUnauthorized},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			req := httptest.NewRequest(sc.method, sc.path, nil)
			if sc.token != "" {
				req.Header.Set("Authorization", "Bearer "+sc.token)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, sc.wantStatus, w.Code)
		})
	}
}
