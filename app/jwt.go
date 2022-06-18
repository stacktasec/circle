package app

import "github.com/golang-jwt/jwt/v4"

type JwtClaims struct {
	TenantID string
	UserType string
	UserRole string
	UserID   string

	jwt.RegisteredClaims
}
