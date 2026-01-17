package cmd

import (
	"vanished-rooms/internal/network"

	"github.com/spf13/cobra"
)

var (
	username  string
	password  string
	publicKey string
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Ejecuta el cliente",
	Run: func(cmd *cobra.Command, args []string) {
		network.StartClient("localhost:8080", username, password, publicKey)
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringVarP(&username, "username", "u", "", "Username for the client")
	clientCmd.Flags().StringVarP(&password, "password", "p", "", "Password for the client")
	clientCmd.MarkFlagRequired("username")
	clientCmd.MarkFlagRequired("password")
}
