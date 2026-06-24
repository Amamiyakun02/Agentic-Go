package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	PBKDF2Iterations = 100000
	KeyLen           = 32 // Length for sha256
)

// HashPassword mengenkripsi password menggunakan PBKDF2-HMAC-SHA256 untuk kompatibilitas dengan Python hashlib
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	key := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeyLen, sha256.New)

	saltHex := hex.EncodeToString(salt)
	keyHex := hex.EncodeToString(key)

	return fmt.Sprintf("pbkdf2:sha256:%d$%s$%s", PBKDF2Iterations, saltHex, keyHex), nil
}

// VerifyPassword memvalidasi string password plaintext terhadap hash PBKDF2
func VerifyPassword(password, hashed string) bool {
	parts := strings.Split(hashed, "$")
	if len(parts) != 3 {
		return false
	}

	// format: algo_info$salt_hex$key_hex
	saltHex := parts[1]
	expectedKeyHex := parts[2]

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return false
	}

	actualKey := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeyLen, sha256.New)
	actualKeyHex := hex.EncodeToString(actualKey)

	return actualKeyHex == expectedKeyHex
}
