package cryptoutils

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(p string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(p), 10)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func VerifyPassword(p, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(p))
	return err == nil
}
