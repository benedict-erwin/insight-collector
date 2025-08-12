package auth

import "github.com/golang-jwt/jwt/v5"

// Claims represents JWT token claims
type Claims struct {
	ClientID string `json:"client_id"` // Random string identifier (only required field)
	jwt.RegisteredClaims
}