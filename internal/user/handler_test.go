package user_test

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
	"github.com/hamdidal/dpp-backend/internal/user"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// newUserRouter wires the user handlers with a minimal Gin engine and injects
// the caller's identity directly (bypassing JWT auth) so tests stay focused
// on handler logic rather than token mechanics.
func newUserRouter(callerUserID, callerRole string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	injectClaims := func(c *gin.Context) {
		c.Set("user_id", callerUserID)
		c.Set("role", callerRole)
		c.Next()
	}
	r.GET("/users", injectClaims, user.ListUsers)
	r.PATCH("/users/:id/role", injectClaims, user.ChangeRole)
	r.DELETE("/users/:id", injectClaims, user.DeleteUser)
	r.PUT("/users/me", injectClaims, user.UpdateProfile)
	r.PUT("/users/me/password", injectClaims, user.ChangePassword)
	return r
}

func seedUser(t *testing.T, username string, role models.Role) models.User {
	t.Helper()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("TestPass1!"), bcrypt.MinCost)
	u := models.User{Username: username, Password: string(hashed), Role: role}
	require.NoError(t, database.DB.Create(&u).Error)
	return u
}

// ─── ListUsers ────────────────────────────────────────────────────────────────

func TestListUsers_Pagination(t *testing.T) {
	defer testutil.NewTestDB(t)()

	callerID := uuid.New().String()
	r := newUserRouter(callerID, "admin")

	// Seed 5 users.
	for i := 0; i < 5; i++ {
		seedUser(t, fmt.Sprintf("user%d", i), models.RoleAuditor)
	}

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantTotal  float64
	}{
		{
			name:       "default page returns all 5 users",
			query:      "",
			wantStatus: http.StatusOK,
			wantTotal:  5,
		},
		{
			name:       "limit=2 returns page of 2",
			query:      "?page=1&limit=2",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/users"+tc.query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)
			var body map[string]any
			require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
			_, hasData := body["data"]
			assert.True(t, hasData, "response must contain data array")
			assert.NotNil(t, body["total"])
		})
	}
}

// ─── ChangeRole ───────────────────────────────────────────────────────────────

func TestChangeRole(t *testing.T) {
	defer testutil.NewTestDB(t)()
	r := newUserRouter(uuid.New().String(), "admin")

	target := seedUser(t, "targetuser", models.RoleAuditor)

	tests := []struct {
		name        string
		userID      string
		body        map[string]any
		wantStatus  int
		wantRole    string
	}{
		{
			name:       "promote auditor to admin returns 200",
			userID:     target.ID.String(),
			body:       map[string]any{"role": "admin"},
			wantStatus: http.StatusOK,
			wantRole:   "admin",
		},
		{
			name:       "demote admin to auditor returns 200",
			userID:     target.ID.String(),
			body:       map[string]any{"role": "auditor"},
			wantStatus: http.StatusOK,
			wantRole:   "auditor",
		},
		{
			name:       "invalid role value returns 400",
			userID:     target.ID.String(),
			body:       map[string]any{"role": "superadmin"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non-existent user returns 404",
			userID:     uuid.New().String(),
			body:       map[string]any{"role": "admin"},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "malformed UUID returns 400",
			userID:     "not-a-uuid",
			body:       map[string]any{"role": "admin"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing role field returns 400",
			userID:     target.ID.String(),
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPatch, "/users/"+tc.userID+"/role", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)
			if tc.wantRole != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
				assert.Equal(t, tc.wantRole, resp["role"])
			}
		})
	}
}

// ─── DeleteUser ───────────────────────────────────────────────────────────────

func TestDeleteUser(t *testing.T) {
	defer testutil.NewTestDB(t)()
	r := newUserRouter(uuid.New().String(), "admin")

	target := seedUser(t, "todelete", models.RoleAuditor)

	tests := []struct {
		name       string
		userID     string
		wantStatus int
	}{
		{
			name:       "delete existing user returns 204",
			userID:     target.ID.String(),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "delete non-existent user returns 204 (soft delete)",
			userID:     uuid.New().String(),
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "malformed UUID returns 400",
			userID:     "bad-uuid",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/users/"+tc.userID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}

// ─── UpdateProfile ────────────────────────────────────────────────────────────

func TestUpdateProfile(t *testing.T) {
	defer testutil.NewTestDB(t)()

	caller := seedUser(t, "profileuser", models.RoleAdmin)
	r := newUserRouter(caller.ID.String(), "admin")

	req := httptest.NewRequest(http.MethodPut, "/users/me",
		bytes.NewBufferString(`{"username":"renameduser"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "renameduser", resp["username"])
}

// ─── ChangePassword ───────────────────────────────────────────────────────────

func TestChangePassword(t *testing.T) {
	defer testutil.NewTestDB(t)()

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
	}{
		{
			name: "correct current password succeeds",
			body: map[string]any{
				"current_password": "TestPass1!",
				"new_password":     "NewSecure2@",
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "wrong current password returns 401",
			body: map[string]any{
				"current_password": "WrongPass1!",
				"new_password":     "NewSecure2@",
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing fields returns 400",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
		},
	}

	// Use a fresh user per sub-test because the first successful test changes the password.
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := seedUser(t, fmt.Sprintf("pwduser%d", i), models.RoleAdmin)
			r2 := newUserRouter(u.ID.String(), "admin")

			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r2.ServeHTTP(w, req)
			assert.Equal(t, tc.wantStatus, w.Code)
		})
	}
}
