package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// UserKey is the context key for storing the authenticated user
type UserKey struct{}

// User represents an authenticated user
type User struct {
	ID       string
	Username string
	Role     string
}

// TokenVerifier defines the interface for token verification
type TokenVerifier interface {
	VerifyToken(token string) (*User, error)
}

// Middleware provides authentication middleware for HTTP requests
type Middleware struct {
	verifier TokenVerifier
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(verifier TokenVerifier) *Middleware {
	return &Middleware{
		verifier: verifier,
	}
}

// Authenticate is a middleware for authenticating HTTP requests
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		token, err := extractToken(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Verify token
		user, err := m.verifier.VerifyToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthenticateUploadRequest is a middleware for tus upload hooks
func (m *Middleware) AuthenticateUploadRequest(r *http.Request) (int, error) {
	// Extract token from Authorization header
	token, err := extractToken(r)
	if err != nil {
		return http.StatusUnauthorized, errors.New("unauthorized")
	}

	// Verify token
	user, err := m.verifier.VerifyToken(token)
	if err != nil {
		return http.StatusUnauthorized, errors.New("unauthorized")
	}

	// Add user to request context
	ctx := context.WithValue(r.Context(), UserKey{}, user)
	*r = *r.WithContext(ctx)

	return http.StatusOK, nil
}

// GetUserFromContext extracts the user from the context
func GetUserFromContext(ctx context.Context) (*User, error) {
	user, ok := ctx.Value(UserKey{}).(*User)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return user, nil
}

// extractToken extracts the token from the Authorization header
func extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is missing")
	}

	// Check for Bearer token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}

// JWTVerifier implements TokenVerifier for JWT tokens
// This is a placeholder - implement actual JWT verification
type JWTVerifier struct {
	secretKey string
}

// NewJWTVerifier creates a new JWT verifier
func NewJWTVerifier(secretKey string) *JWTVerifier {
	return &JWTVerifier{
		secretKey: secretKey,
	}
}

// VerifyToken verifies a JWT token
// Note: This is a simplified placeholder. In a real implementation, use a proper JWT library
func (v *JWTVerifier) VerifyToken(token string) (*User, error) {
	// In a real implementation, parse and verify the JWT token
	// For this example, we'll just return a mock user
	// In a production environment, use a proper JWT library like github.com/golang-jwt/jwt

	// Placeholder implementation
	if token == "" {
		return nil, errors.New("invalid token")
	}

	// Mock user for this example
	return &User{
		ID:       "user-123",
		Username: "testuser",
		Role:     "user",
	}, nil
}
 
