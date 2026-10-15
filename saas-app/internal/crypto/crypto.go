// Package crypto provides password hashing and token generation using only
// the Go standard library.
//
// NOTE: this is a demonstration-grade implementation so the vertical slice runs
// with zero external dependencies. For production, swap HashPassword/Verify
// for argon2id or bcrypt (e.g. golang.org/x/crypto).
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	saltSize   = 16
	iterations = 100_000
	keyLen     = 32
)

// GenerateToken returns a URL-safe random token.
func GenerateToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// GenerateCode returns an unguessable invitation code.
func GenerateCode() string {
	return GenerateToken()
}

// HashPassword derives a salted, iterated SHA-256 key and encodes it as
// `salted-sha256$<saltB64>$<iter>$<keyHex>`.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := deriveKey([]byte(password), salt, iterations, keyLen)
	return fmt.Sprintf("salted-sha256$%s$%d$%s",
		base64.StdEncoding.EncodeToString(salt), iterations, hex.EncodeToString(key)), nil
}

// VerifyPassword checks a plaintext password against a hash from HashPassword.
func VerifyPassword(hash, password string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != "salted-sha256" {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	expectHex := parts[3]
	key := deriveKey([]byte(password), salt, iterations, keyLen)
	gotHex := hex.EncodeToString(key)
	return subtle.ConstantTimeCompare([]byte(gotHex), []byte(expectHex)) == 1
}

func deriveKey(password, salt []byte, iter, keyLen int) []byte {
	x := append(append([]byte{}, salt...), password...)
	out := make([]byte, keyLen)
	h := sha256.New()
	for i := 0; i < iter; i++ {
		h.Reset()
		h.Write(x)
		x = h.Sum(nil)
	}
	copy(out, x)
	return out
}