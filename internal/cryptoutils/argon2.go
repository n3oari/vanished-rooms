package cryptoutils

import (
	"crypto/rand"
	"crypto/subtle"

	"golang.org/x/crypto/argon2"
)

const (
	Memory      = 64 * 1024
	Iterations  = 2
	Parallelism = 4
	KeyLength   = 32
)

func GenerarSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}

func HashPassword(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, Iterations, Memory, Parallelism, KeyLength)
}

func VerifyPassword(password string, salt []byte, originalHash []byte) bool {
	newHash := argon2.IDKey([]byte(password), salt, Iterations, Memory, Parallelism, KeyLength)
	return subtle.ConstantTimeCompare(newHash, originalHash) == 1
}
