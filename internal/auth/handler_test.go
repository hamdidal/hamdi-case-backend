package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hamdidal/dpp-backend/internal/models"
	"github.com/hamdidal/dpp-backend/internal/testutil"
	"github.com/hamdidal/dpp-backend/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ─── Pure-Function Tests ──────────────────────────────────────────────────────

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  string
	}{
		{
			name:    "too short (7 chars)",
			password: "Ab1!xyz",
			wantErr: "password must be at least 8 characters",
		},
		{
			name:    "exactly 8 chars but no uppercase",
			password: "ab1!aaaa",
			wantErr: "password must contain at least 1 uppercase letter",
		},
		{
			name:    "no digit",
			password: "Abcdefg!",
			wantErr: "password must contain at least 1 digit",
		},
		{
			name:    "no special character",
			password: "Abcdefg1",
			wantErr: "password must contain at least 1 special character (!@#$%^&*)",
		},
		{
			name:    "valid: minimum viable password",
			password: "Abcdef1!",
			wantErr: "",
		},
		{
			name:    "valid: all allowed specials",
			password: "Pass1@#$%^&*",
			wantErr: "",
		},
		{
			name:    "valid: long password",
			password: "SuperSecure1!Pass2024",
			wantErr: "",
		},
		{
			name:    "empty string",
			password: "",
			wantErr: "password must be at least 8 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validatePassword(tc.password)
			assert.Equal(t, tc.wantErr, got)
		})
	}
}

func TestUsernameRegex(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		wantMatch bool
	}{
		{name: "alphanumeric only", username: "john123", wantMatch: true},
		{name: "dots allowed", username: "john.doe", wantMatch: true},
		{name: "underscores allowed", username: "john_doe", wantMatch: true},
		{name: "exactly 3 characters", username: "abc", wantMatch: true},
		{name: "exactly 24 characters", username: strings.Repeat("a", 24), wantMatch: true},
		{name: "mixed all allowed chars", username: "j0hn.doe_99", wantMatch: true},
		{name: "too short (2 chars)", username: "ab", wantMatch: false},
		{name: "too long (25 chars)", username: strings.Repeat("a", 25), wantMatch: false},
		{name: "hyphen not allowed", username: "john-doe", wantMatch: false},
		{name: "space not allowed", username: "john doe", wantMatch: false},
		{name: "at-sign not allowed", username: "john@doe", wantMatch: false},
		{name: "empty string", username: "", wantMatch: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantMatch, usernameRe.MatchString(tc.username))
		})
	}
}

// ─── Handler Integration Tests (SQLite in-memory) ────────────────────────────

func newAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", Login)
	r.POST("/register", Register)
	return r
}

func seedTestUser(t *testing.T, username, plainPassword string, role models.Role) models.User {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.MinCost)
	require.NoError(t, err, "bcrypt hash failed")
	u := models.User{Username: username, Password: string(hashed), Role: role}
	require.NoError(t, database.DB.Create(&u).Error)
	return u
}

func TestLogin_Handler(t *testing.T) {
	defer testutil.NewTestDB(t)()
	t.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")

	seedTestUser(t, "testadmin", "TestPass1!", models.RoleAdmin)

	r := newAuthRouter()

	tests := []struct {
		name        string
		body        map[string]any
		wantStatus  int
		wantToken   bool
		wantErrKey  string
	}{
		{
			name:       "valid admin credentials returns 200 with token",
			body:       map[string]any{"username": "testadmin", "password": "TestPass1!"},
			wantStatus: http.StatusOK,
			wantToken:  true,
		},
		{
			name:       "wrong password returns 401",
			body:       map[string]any{"username": "testadmin", "password": "WrongPass1!"},
			wantStatus: http.StatusUnauthorized,
			wantErrKey: "error",
		},
		{
			name:       "non-existent user returns 401",
			body:       map[string]any{"username": "ghost", "password": "TestPass1!"},
			wantStatus: http.StatusUnauthorized,
			wantErrKey: "error",
		},
		{
			name:       "missing username field returns 400",
			body:       map[string]any{"password": "TestPass1!"},
			wantStatus: http.StatusBadRequest,
			wantErrKey: "error",
		},
		{
			name:       "missing password field returns 400",
			body:       map[string]any{"username": "testadmin"},
			wantStatus: http.StatusBadRequest,
			wantErrKey: "error",
		},
		{
			name:       "empty JSON object returns 400",
			body:       map[string]any{},
			wantStatus: http.StatusBadRequest,
			wantErrKey: "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.wantStatus, w.Code)

			var resp map[string]any
			require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

			if tc.wantToken {
				token, ok := resp["token"].(string)
				assert.True(t, ok, "response must contain a string token")
				assert.NotEmpty(t, token)
			}
			if tc.wantErrKey != "" {
				assert.Contains(t, resp, tc.wantErrKey, "response must contain an error field")
			}
		})
	}
}

func TestRegister_Handler(t *testing.T) {
	defer testutil.NewTestDB(t)()

	r := newAuthRouter()

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantRole   string
	}{
		{
			name:       "valid registration defaults role to auditor",
			body:       map[string]any{"username": "newauditor", "password": "SecurePass1!"},
			wantStatus: http.StatusCreated,
			wantRole:   "auditor",
		},
		{
			name:       "explicit auditor role accepted",
			body:       map[string]any{"username": "auditor2", "password": "SecurePass1!", "role": "auditor"},
			wantStatus: http.StatusCreated,
			wantRole:   "auditor",
		},
		{
			name:       "explicit admin role accepted",
			body:       map[string]any{"username": "adminuser", "password": "SecurePass1!", "role": "admin"},
			wantStatus: http.StatusCreated,
			wantRole:   "admin",
		},
		{
			name:       "weak password — too short",
			body:       map[string]any{"username": "weakuser", "password": "Abc1!"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "weak password — no uppercase",
			body:       map[string]any{"username": "weakuser2", "password": "abcdefg1!"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "weak password — no digit",
			body:       map[string]any{"username": "weakuser3", "password": "Abcdefgh!"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "weak password — no special character",
			body:       map[string]any{"username": "weakuser4", "password": "Abcdefg1"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid username — too short",
			body:       map[string]any{"username": "ab", "password": "SecurePass1!"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid username — contains hyphen",
			body:       map[string]any{"username": "bad-name", "password": "SecurePass1!"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid role value",
			body:       map[string]any{"username": "validuser", "password": "SecurePass1!", "role": "superuser"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing username returns 400",
			body:       map[string]any{"password": "SecurePass1!"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
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

func TestRegister_DuplicateUsername(t *testing.T) {
	defer testutil.NewTestDB(t)()

	r := newAuthRouter()
	body := `{"username":"dupeuser","password":"SecurePass1!"}`

	// First registration must succeed.
	req1 := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Second registration with the same username must return 409.
	req2 := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w2.Body).Decode(&resp))
	assert.Contains(t, resp, "error")
}
