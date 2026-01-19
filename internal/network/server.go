package network

import (
	"log"
	"net"
	"sync"
	"vanished-rooms/internal/storage"
	"vanished-rooms/internal/ui"

	"github.com/google/uuid"
)

type Server struct {
	Clients          map[string]net.Conn
	UsersInRoom      map[string]*storage.Users
	mu               sync.Mutex
	SQLiteRepository *storage.SQLiteRepository
}

func StartServer(port string, repository *storage.SQLiteRepository) {
	ui.PrintRandomBanner()
	sv := &Server{
		Clients:          make(map[string]net.Conn),
		UsersInRoom:      make(map[string]*storage.Users),
		SQLiteRepository: repository,
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	log.Printf("[+] Vanished Rooms server listening on port: %s...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go sv.HandleConnection(conn)
	}
}

func generateUUID() string {
	return uuid.New().String()
}
