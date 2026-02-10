package cmd

import (
	"database/sql"
	"log"
	"vanished-rooms/internal/network"
	"vanished-rooms/internal/storage"

	_ "modernc.org/sqlite"

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
		db, err := sql.Open("sqlite", `./internal/storage/sqlite.db`)
		if err != nil {
			log.Fatalf("[-] Error to open SQLite: %v", err)
		}
		defer db.Close()

		if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
			log.Fatalf("[-] Failed to set WAL mode: %v", err)
		}
		if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
			log.Fatalf("[-] Failed to enable Foreign Keys: %v", err)
		}

		if err := storage.InitDB(db); err != nil {
			log.Fatalf("[-] Failed to initialize database: %v", err)
		}

		repository := storage.NewSQLHandler(db)

		network.StartServer(port, repository)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on for incoming TCP connections")
}
