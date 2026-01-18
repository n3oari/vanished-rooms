package cryptoutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
)

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no se encontraron datos PEM válidos")
	}

	if block.Type == "PRIVATE KEY" {
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if priv, ok := key.(*rsa.PrivateKey); ok {
			return priv, nil
		}
		return nil, fmt.Errorf("la clave privada no es de tipo RSA")
	}
	if block.Type == "RSA PRIVATE KEY" {
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}

	return nil, fmt.Errorf("tipo de bloque PEM no soportado: %s", block.Type)
}

func GetPublicKeyPEM(priv *rsa.PrivateKey) (string, error) {
	pub := &priv.PublicKey
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}

	return string(pem.EncodeToMemory(block)), nil

}

func EncodePublicKeyToBase64(priv *rsa.PrivateKey) (string, error) {
	pemString, err := GetPublicKeyPEM(priv)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString([]byte(pemString)), nil
}

func EncryptWithPublicKey(data []byte, pubKeyBase64 string) ([]byte, error) {
	pemBytes, err := base64.StdEncoding.DecodeString(pubKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("error decodificando base64: %v", err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("error al decodificar llave pública")
	}

	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pubKey, ok := pubInterface.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("no es una llave RSA válida")
	}

	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, data, nil)
}

func DecryoptWithPrivateKey(ciphertextB64 string, privKey *rsa.PrivateKey) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("error decodificando base64: %v", err)
	}

	plaintext, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		privKey,
		ciphertext,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error descifrando con RSA: %v", err)
	}
	return plaintext, nil
}
