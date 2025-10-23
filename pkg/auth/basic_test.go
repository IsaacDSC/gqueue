package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBasicAuth(t *testing.T) {
	t.Run("with users map", func(t *testing.T) {
		users := map[string]string{
			"admin": "password123",
			"user":  "secret",
		}
		auth := NewBasicAuth(users)
		assert.NotNil(t, auth)
		assert.Equal(t, users, auth.users)
	})

	t.Run("with nil users map", func(t *testing.T) {
		auth := NewBasicAuth(nil)
		assert.NotNil(t, auth)
		assert.NotNil(t, auth.users)
		assert.Len(t, auth.users, 0)
	})
}

func TestBasicAuth_AddUser(t *testing.T) {
	auth := NewBasicAuth(nil)
	auth.AddUser("testuser", "testpass")

	assert.Equal(t, "testpass", auth.users["testuser"])
}

func TestBasicAuth_RemoveUser(t *testing.T) {
	users := map[string]string{
		"admin": "password123",
		"user":  "secret",
	}
	auth := NewBasicAuth(users)

	auth.RemoveUser("user")

	_, exists := auth.users["user"]
	assert.False(t, exists)
	assert.Equal(t, "password123", auth.users["admin"])
}

func TestBasicAuth_ValidateCredentials(t *testing.T) {
	users := map[string]string{
		"admin": "password123",
		"user":  "secret",
	}
	auth := NewBasicAuth(users)

	t.Run("success - valid credentials", func(t *testing.T) {
		valid := auth.ValidateCredentials("admin", "password123")
		assert.True(t, valid)
	})

	t.Run("error - invalid password", func(t *testing.T) {
		valid := auth.ValidateCredentials("admin", "wrongpassword")
		assert.False(t, valid)
	})

	t.Run("error - user not found", func(t *testing.T) {
		valid := auth.ValidateCredentials("nonexistent", "password")
		assert.False(t, valid)
	})

	t.Run("error - empty credentials", func(t *testing.T) {
		valid := auth.ValidateCredentials("", "")
		assert.False(t, valid)
	})
}

func TestBasicAuth_ParseBasicAuth(t *testing.T) {
	auth := NewBasicAuth(nil)

	t.Run("success - valid basic auth header", func(t *testing.T) {
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))
		authHeader := "Basic " + credentials

		username, password, err := auth.ParseBasicAuth(authHeader)

		assert.NoError(t, err)
		assert.Equal(t, "admin", username)
		assert.Equal(t, "password123", password)
	})

	t.Run("success - credentials with colon in password", func(t *testing.T) {
		credentials := base64.StdEncoding.EncodeToString([]byte("user:pass:word"))
		authHeader := "Basic " + credentials

		username, password, err := auth.ParseBasicAuth(authHeader)

		assert.NoError(t, err)
		assert.Equal(t, "user", username)
		assert.Equal(t, "pass:word", password)
	})

	t.Run("error - missing authorization header", func(t *testing.T) {
		username, password, err := auth.ParseBasicAuth("")

		assert.Error(t, err)
		assert.Equal(t, ErrMissingAuthHeader, err)
		assert.Empty(t, username)
		assert.Empty(t, password)
	})

	t.Run("error - unsupported auth type", func(t *testing.T) {
		authHeader := "Bearer token123"

		username, password, err := auth.ParseBasicAuth(authHeader)

		assert.Error(t, err)
		assert.Equal(t, ErrUnsupportedAuthType, err)
		assert.Empty(t, username)
		assert.Empty(t, password)
	})

	t.Run("error - invalid base64", func(t *testing.T) {
		authHeader := "Basic invalid_base64!"

		username, password, err := auth.ParseBasicAuth(authHeader)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidAuthHeader, err)
		assert.Empty(t, username)
		assert.Empty(t, password)
	})

	t.Run("error - missing credentials", func(t *testing.T) {
		authHeader := "Basic "

		username, password, err := auth.ParseBasicAuth(authHeader)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidAuthHeader, err)
		assert.Empty(t, username)
		assert.Empty(t, password)
	})

	t.Run("error - no colon separator", func(t *testing.T) {
		credentials := base64.StdEncoding.EncodeToString([]byte("adminpassword"))
		authHeader := "Basic " + credentials

		username, password, err := auth.ParseBasicAuth(authHeader)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidAuthHeader, err)
		assert.Empty(t, username)
		assert.Empty(t, password)
	})
}

func TestBasicAuth_Authenticate(t *testing.T) {
	users := map[string]string{
		"admin": "password123",
		"user":  "secret",
	}
	auth := NewBasicAuth(users)

	t.Run("success - valid authentication", func(t *testing.T) {
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))
		authHeader := "Basic " + credentials

		username, err := auth.Authenticate(authHeader)

		assert.NoError(t, err)
		assert.Equal(t, "admin", username)
	})

	t.Run("error - invalid credentials", func(t *testing.T) {
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:wrongpassword"))
		authHeader := "Basic " + credentials

		username, err := auth.Authenticate(authHeader)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Empty(t, username)
	})

	t.Run("error - user not found", func(t *testing.T) {
		credentials := base64.StdEncoding.EncodeToString([]byte("nonexistent:password"))
		authHeader := "Basic " + credentials

		username, err := auth.Authenticate(authHeader)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Empty(t, username)
	})

	t.Run("error - parsing error", func(t *testing.T) {
		authHeader := "Bearer token123"

		username, err := auth.Authenticate(authHeader)

		assert.Error(t, err)
		assert.Equal(t, ErrUnsupportedAuthType, err)
		assert.Empty(t, username)
	})
}

func TestBasicAuth_Middleware(t *testing.T) {
	users := map[string]string{
		"admin": "password123",
	}
	auth := NewBasicAuth(users)

	// Mock handler that should only be called if auth succeeds
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	middleware := auth.Middleware(handler)

	t.Run("success - valid credentials", func(t *testing.T) {
		handlerCalled = false
		var authenticatedUser string
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))

		// Update handler to check context
		handler := func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			username, ok := UserFromContext(r.Context())
			if ok {
				authenticatedUser = username
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}
		middleware := auth.Middleware(handler)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		w := httptest.NewRecorder()

		middleware(w, req)

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "success", w.Body.String())
		assert.Equal(t, "admin", authenticatedUser)
	})

	t.Run("error - missing authorization", func(t *testing.T) {
		handlerCalled = false

		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		middleware(w, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic realm")
	})

	t.Run("error - invalid credentials", func(t *testing.T) {
		handlerCalled = false
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:wrongpassword"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		w := httptest.NewRecorder()

		middleware(w, req)

		assert.False(t, handlerCalled)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic realm")
	})
}

func TestBasicAuth_RequireAuth(t *testing.T) {
	users := map[string]string{
		"admin": "password123",
	}
	auth := NewBasicAuth(users)

	t.Run("success - valid credentials", func(t *testing.T) {
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		w := httptest.NewRecorder()

		username, ok := auth.RequireAuth(w, req)

		assert.True(t, ok)
		assert.Equal(t, "admin", username)
		assert.Equal(t, http.StatusOK, w.Code) // No error response written
	})

	t.Run("error - invalid credentials", func(t *testing.T) {
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:wrongpassword"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		w := httptest.NewRecorder()

		username, ok := auth.RequireAuth(w, req)

		assert.False(t, ok)
		assert.Empty(t, username)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic realm")
	})

	t.Run("error - missing authorization", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		username, ok := auth.RequireAuth(w, req)

		assert.False(t, ok)
		assert.Empty(t, username)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Header().Get("WWW-Authenticate"), "Basic realm")
	})
}

func TestBasicAuth_MiddlewareContextIntegration(t *testing.T) {
	users := map[string]string{
		"admin": "password123",
		"user":  "secret",
	}
	auth := NewBasicAuth(users)

	t.Run("context contains correct username", func(t *testing.T) {
		var contextUsername string
		var hasUser bool

		handler := func(w http.ResponseWriter, r *http.Request) {
			contextUsername, hasUser = UserFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}

		middleware := auth.Middleware(handler)
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		w := httptest.NewRecorder()

		middleware(w, req)

		assert.True(t, hasUser)
		assert.Equal(t, "admin", contextUsername)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("context works with different users", func(t *testing.T) {
		var contextUsername string

		handler := func(w http.ResponseWriter, r *http.Request) {
			contextUsername, _ = UserFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}

		middleware := auth.Middleware(handler)
		credentials := base64.StdEncoding.EncodeToString([]byte("user:secret"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		w := httptest.NewRecorder()

		middleware(w, req)

		assert.Equal(t, "user", contextUsername)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("MustUserFromContext works in middleware", func(t *testing.T) {
		var contextUsername string
		var panicked bool

		handler := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					panicked = true
				}
			}()
			contextUsername = MustUserFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}

		middleware := auth.Middleware(handler)
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		w := httptest.NewRecorder()

		middleware(w, req)

		assert.False(t, panicked)
		assert.Equal(t, "admin", contextUsername)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("HasUser returns true in middleware", func(t *testing.T) {
		var hasUser bool

		handler := func(w http.ResponseWriter, r *http.Request) {
			hasUser = HasUser(r.Context())
			w.WriteHeader(http.StatusOK)
		}

		middleware := auth.Middleware(handler)
		credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic "+credentials)
		w := httptest.NewRecorder()

		middleware(w, req)

		assert.True(t, hasUser)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// Benchmark tests
func BenchmarkBasicAuth_ValidateCredentials(b *testing.B) {
	users := map[string]string{
		"admin": "password123",
	}
	auth := NewBasicAuth(users)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.ValidateCredentials("admin", "password123")
	}
}

func BenchmarkBasicAuth_ParseBasicAuth(b *testing.B) {
	auth := NewBasicAuth(nil)
	credentials := base64.StdEncoding.EncodeToString([]byte("admin:password123"))
	authHeader := "Basic " + credentials

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.ParseBasicAuth(authHeader)
	}
}
