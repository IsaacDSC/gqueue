package auth

import (
	"context"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserContextKey is the key used to store the authenticated username in context
	UserContextKey ContextKey = "authenticated_user"
)

// WithUser adds the authenticated username to the context
func WithUser(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, UserContextKey, username)
}

// UserFromContext retrieves the authenticated username from the context
// Returns the username and a boolean indicating if it was found
func UserFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(UserContextKey).(string)
	return username, ok
}

// MustUserFromContext retrieves the authenticated username from the context
// Panics if the username is not found in the context
func MustUserFromContext(ctx context.Context) string {
	username, ok := UserFromContext(ctx)
	if !ok {
		panic("authenticated user not found in context")
	}
	return username
}

// HasUser checks if there is an authenticated user in the context
func HasUser(ctx context.Context) bool {
	_, ok := UserFromContext(ctx)
	return ok
}
