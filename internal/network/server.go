package network

import (
	"log"
	"net"
	"sync"
	"vanished-rooms/internal/storage"
	"vanished-rooms/internal/ui"

	"github.com/google/uuid"
)

type ClientSession struct {
	Conn      net.Conn
	ID        string
	Username  string
	PublicKey string
	Room      string
}

type Server struct {
	Clients          map[string]*ClientSession
	mu               sync.RWMutex // RWMutex es mejor para lecturas concurrentes
	SQLiteRepository *storage.SQLiteRepository
}

func StartServer(port string, repository *storage.SQLiteRepository) {
	ui.PrintRandomBanner()

	sv := &Server{
		Clients:          make(map[string]*ClientSession),
		SQLiteRepository: repository,
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Println("[-] Error starting server:", err)
		return
	}

	defer listener.Close()

	log.Printf("[+] Vanished Rooms server listening locally on: 127.0.0.1:%s (Protected by Tor)\n", port)

	log.Printf("[+] Vanished Rooms server listening on port: %s...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("[-] Error accepting connection:", err)
			continue
		}

		go sv.HandleConnection(conn)
	}
}

func generateUUID() string {
	return uuid.New().String()
}
