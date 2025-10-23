package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrMissingAuthHeader   = errors.New("missing authorization header")
	ErrInvalidAuthHeader   = errors.New("invalid authorization header format")
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrUnsupportedAuthType = errors.New("unsupported authorization type")
)

// BasicAuth represents a basic authentication handler
type BasicAuth struct {
	users map[string]string // username -> password mapping
}

// NewBasicAuth creates a new BasicAuth instance with the provided users
func NewBasicAuth(users map[string]string) *BasicAuth {
	if users == nil {
		users = make(map[string]string)
	}
	return &BasicAuth{
		users: users,
	}
}

// AddUser adds a user to the authentication system
func (ba *BasicAuth) AddUser(username, password string) {
	ba.users[username] = password
}

// RemoveUser removes a user from the authentication system
func (ba *BasicAuth) RemoveUser(username string) {
	delete(ba.users, username)
}

// ValidateCredentials validates the provided username and password
func (ba *BasicAuth) ValidateCredentials(username, password string) bool {
	storedPassword, exists := ba.users[username]
	if !exists {
		return false
	}

	// Use constant time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(password), []byte(storedPassword)) == 1
}

// ParseBasicAuth parses the Authorization header and extracts username and password
func (ba *BasicAuth) ParseBasicAuth(authHeader string) (username, password string, err error) {
	if authHeader == "" {
		return "", "", ErrMissingAuthHeader
	}

	// Check if it starts with "Basic "
	const basicPrefix = "Basic "
	if !strings.HasPrefix(authHeader, basicPrefix) {
		return "", "", ErrUnsupportedAuthType
	}

	// Extract the base64 encoded part
	encoded := strings.TrimPrefix(authHeader, basicPrefix)
	if encoded == "" {
		return "", "", ErrInvalidAuthHeader
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", ErrInvalidAuthHeader
	}

	// Split username:password
	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return "", "", ErrInvalidAuthHeader
	}

	return parts[0], parts[1], nil
}

// Authenticate validates the Authorization header from an HTTP request
func (ba *BasicAuth) Authenticate(authHeader string) (string, error) {
	username, password, err := ba.ParseBasicAuth(authHeader)
	if err != nil {
		return "", err
	}

	if !ba.ValidateCredentials(username, password) {
		return "", ErrInvalidCredentials
	}

	return username, nil
}

// Middleware returns an HTTP middleware that validates Basic Auth
func (ba *BasicAuth) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		username, err := ba.Authenticate(authHeader)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add username to request context for use in handlers
		ctx := WithUser(r.Context(), username)
		r = r.WithContext(ctx)
		next(w, r)
	}
}

// RequireAuth is a helper function to quickly check authentication in handlers
func (ba *BasicAuth) RequireAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	username, err := ba.Authenticate(authHeader)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}
	return username, true
}
