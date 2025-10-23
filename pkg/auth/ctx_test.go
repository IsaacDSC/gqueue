package auth

import (
	"context"
	"testing"
)

func TestWithUser(t *testing.T) {
	ctx := context.Background()
	username := "testuser"

	newCtx := WithUser(ctx, username)

	if newCtx == ctx {
		t.Error("WithUser should return a new context")
	}

	retrievedUsername, ok := UserFromContext(newCtx)
	if !ok {
		t.Error("Expected to find username in context")
	}

	if retrievedUsername != username {
		t.Errorf("Expected username %s, got %s", username, retrievedUsername)
	}
}

func TestUserFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected string
		found    bool
	}{
		{
			name: "context with user",
			setup: func() context.Context {
				return WithUser(context.Background(), "testuser")
			},
			expected: "testuser",
			found:    true,
		},
		{
			name: "context without user",
			setup: func() context.Context {
				return context.Background()
			},
			expected: "",
			found:    false,
		},
		{
			name: "context with empty username",
			setup: func() context.Context {
				return WithUser(context.Background(), "")
			},
			expected: "",
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			username, found := UserFromContext(ctx)

			if found != tt.found {
				t.Errorf("Expected found=%v, got found=%v", tt.found, found)
			}

			if username != tt.expected {
				t.Errorf("Expected username=%s, got username=%s", tt.expected, username)
			}
		})
	}
}

func TestMustUserFromContext(t *testing.T) {
	t.Run("context with user", func(t *testing.T) {
		ctx := WithUser(context.Background(), "testuser")
		username := MustUserFromContext(ctx)

		if username != "testuser" {
			t.Errorf("Expected username=testuser, got username=%s", username)
		}
	})

	t.Run("context without user should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected MustUserFromContext to panic")
			}
		}()

		ctx := context.Background()
		MustUserFromContext(ctx)
	})
}

func TestHasUser(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected bool
	}{
		{
			name: "context with user",
			setup: func() context.Context {
				return WithUser(context.Background(), "testuser")
			},
			expected: true,
		},
		{
			name: "context without user",
			setup: func() context.Context {
				return context.Background()
			},
			expected: false,
		},
		{
			name: "context with empty username",
			setup: func() context.Context {
				return WithUser(context.Background(), "")
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			hasUser := HasUser(ctx)

			if hasUser != tt.expected {
				t.Errorf("Expected hasUser=%v, got hasUser=%v", tt.expected, hasUser)
			}
		})
	}
}

func TestContextKeyType(t *testing.T) {
	// Test that our custom context key type works correctly
	ctx := context.Background()

	// Add user using our functions
	ctx = WithUser(ctx, "testuser")

	// Try to access with the raw key (should work)
	value := ctx.Value(UserContextKey)
	if value != "testuser" {
		t.Errorf("Expected value=testuser, got value=%v", value)
	}

	// Try to access with a string key (should not work due to type safety)
	value = ctx.Value("authenticated_user")
	if value != nil {
		t.Error("Expected nil when accessing with string key, but got a value")
	}
}

func TestMultipleUsersInDifferentContexts(t *testing.T) {
	ctx1 := WithUser(context.Background(), "user1")
	ctx2 := WithUser(context.Background(), "user2")

	username1, ok1 := UserFromContext(ctx1)
	username2, ok2 := UserFromContext(ctx2)

	if !ok1 || !ok2 {
		t.Error("Expected to find usernames in both contexts")
	}

	if username1 != "user1" {
		t.Errorf("Expected username1=user1, got username1=%s", username1)
	}

	if username2 != "user2" {
		t.Errorf("Expected username2=user2, got username2=%s", username2)
	}

	// Ensure contexts are independent
	if username1 == username2 {
		t.Error("Contexts should be independent")
	}
}
