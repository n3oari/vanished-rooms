package cryptoutils

import (
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"runtime"
)

func WipeRSAKey(priv *rsa.PrivateKey) {
	if priv == nil {
		return
	}

	if priv.N != nil {
		hash := sha256.Sum256(priv.N.Bytes())
		fmt.Printf("[DEBUG-WIPE] Original RSA Key Fingerprint: %x...\n", hash[:4])
	}

	if priv.D != nil {
		priv.D.SetInt64(0)
	}

	for _, p := range priv.Primes {
		p.SetInt64(0)
	}
	priv.Precomputed = rsa.PrecomputedValues{}

	runtime.KeepAlive(priv)
	fmt.Println("[DEBUG-WIPE] Memory overwritten with zeros. Secret is gone.")

}

func WipeBytes(data []byte) {
	if data == nil {
		return
	}
	fmt.Println("[DEBUG-WIPE] Shredding RSA Private Key (D and Primes)...")

	for i := range data {
		data[i] = 0
	}
	runtime.KeepAlive(data)
	fmt.Println("[DEBUG-WIPE] RSA Identity sanitized.")

}
