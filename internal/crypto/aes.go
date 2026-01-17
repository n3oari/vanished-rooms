package crypto

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateAESKey() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil

}
