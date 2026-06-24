package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GetJWTSecret mengambil secret key dari environment
func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super-secret-jwt-key-aimer-future-2026-06-02"
	}
	return []byte(secret)
}

// CreateAccessToken membuat JWT token HMAC SHA-256 untuk sesi login
func CreateAccessToken(data map[string]interface{}) (string, error) {
	claims := jwt.MapClaims{}
	for k, v := range data {
		claims[k] = v
	}

	// Set expiration time (default 7 hari jika tidak dispesifikasikan)
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(7 * 24 * time.Hour).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetJWTSecret())
}
