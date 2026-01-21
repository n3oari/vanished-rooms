package cmd

import (
	"crypto/rsa"
	"fmt"
	"log"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/network"

	"github.com/spf13/cobra"
)

var (
	username       string
	password       string
	privateKeyPath string
	proxyAddr      string
	tor            bool
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run as client and register",
	Run: func(cmd *cobra.Command, args []string) {
		priv, pubKeyToSend := prepareKeys(privateKeyPath)

		if tor {
			fmt.Println("[TOR] Trying to connect through onion protocol in 127.0.0.1:9050...")
		} else if proxyAddr != "" {
			fmt.Printf("[PROXY] Routing traffic through %s (Burp Suite)\n", proxyAddr)
		} else {
			fmt.Println("[STANDARD] Normal connection")
		}
		network.StartClient("localhost:8080", username, password, pubKeyToSend, tor, proxyAddr, priv)
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringVarP(&username, "username", "u", "", "Username for the client")
	clientCmd.Flags().StringVarP(&password, "password", "p", "", "Password for the client")
	clientCmd.Flags().StringVarP(&privateKeyPath, "key", "k", "", "Path to your RSA private key (.pem)")
	clientCmd.Flags().BoolVarP(&tor, "tor", "t", false, "Use TOR for the connection")
	clientCmd.Flags().StringVarP(&proxyAddr, "proxy", "x", "", "HTTP proxy address (e.g., 127.0.0.1:8080)")
	clientCmd.MarkFlagRequired("username")
	clientCmd.MarkFlagRequired("password")
	clientCmd.MarkFlagRequired("key")
}

func prepareKeys(path string) (*rsa.PrivateKey, string) {
	privKey, err := cryptoutils.LoadPrivateKey(path)
	if err != nil {
		log.Fatalf("Error loading private key: %v", err)
	}

	pubKeyToSend, err := cryptoutils.EncodePublicKeyToBase64(privKey)
	if err != nil {
		log.Fatalf("Error generating a public key: %v", err)
	}

	return privKey, pubKeyToSend
}
