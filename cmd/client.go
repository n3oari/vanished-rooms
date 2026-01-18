package cmd

import (
	"crypto/rsa"
	"log"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/network"

	"github.com/spf13/cobra"
)

var (
	username       string
	password       string
	privateKeyPath string
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Run as client and register",
	Run: func(cmd *cobra.Command, args []string) {
		priv, _ := prepareKeys(privateKeyPath)
		network.MyPrivateKey = priv
		network.StartClient("localhost:8080", username, password, privateKeyPath)
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringVarP(&username, "username", "u", "", "Username for the client")
	clientCmd.Flags().StringVarP(&password, "password", "p", "", "Password for the client")
	clientCmd.Flags().StringVarP(&privateKeyPath, "key", "k", "", "Path to your RSA private key (.pem)")
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
