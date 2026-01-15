package cmd

import (
	"vanished-rooms/internal/network"

	"github.com/spf13/cobra"
)

var (
	port string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the vanished-rooms messaging server",
	Long:  `Launches a TCP server to handle message broadcasting between connected clients.`,
	Run: func(cmd *cobra.Command, args []string) {
		network.StartServer(port)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on for incoming TCP connections")
}
