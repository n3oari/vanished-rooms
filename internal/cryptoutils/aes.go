package cryptoutils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func GenerateAESKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func Encrypt(plaintext []byte, key []byte) (payload []byte, iv []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	iv = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	payload = gcm.Seal(nil, iv, plaintext, nil)
	return payload, iv, nil
}

func EncryptForChat(plaintText string, key []byte) (string, error) {
	payload, iv, err := Encrypt([]byte(plaintText), key)
	if err != nil {
		return "", err
	}

	combined := append(iv, payload...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

func Decrypt(payload []byte, iv []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, iv, payload, nil)
}

func DecryptForChat(cipherTextB64 string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cipherTextB64)
	if err != nil {
		return "", err
	}
	// AES-GCM always use a Noncne of 12 bytes
	ivSize := 12
	if len(data) < ivSize {
		return "", fmt.Errorf("el mensaje es demasiado corto")
	}

	iv := data[:ivSize]
	payload := data[ivSize:]

	plaintext, err := Decrypt(payload, iv, key)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
