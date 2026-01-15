package cmd

import (
	"vanished-rooms/internal/network" // Importas tu lógica interna

	"github.com/spf13/cobra"
)

var (
	username string
	password string
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Ejecuta el cliente",
	Run: func(cmd *cobra.Command, args []string) {
		// Llamas a la función del OTRO client.go (el de internal)
		network.StartClient("localhost:8080", username, password)
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringVarP(&username, "username", "u", "", "Nombre de usuario para el cliente")
	clientCmd.Flags().StringVarP(&password, "password", "p", "", "Contraseña para el cliente")
	clientCmd.MarkFlagRequired("username")
	clientCmd.MarkFlagRequired("password")

}
