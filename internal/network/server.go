package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"

	"vanished-rooms/internal/ui"

	"vanished-rooms/internal/storage"

	"github.com/google/uuid"
)

type Server struct {
	clients          map[string]net.Conn
	mu               sync.Mutex
	SQLiteRepository *storage.SQLiteRepository
}

func StartServer(port string, repository *storage.SQLiteRepository) {
	ui.PrintBanner2()
	sv := &Server{
		clients:          make(map[string]net.Conn),
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
		go sv.handleConnection(conn)
	}
}

// func (r *SQLiteRepository) CreateUser(u Users) error {
//	query := `INSERT INTO users (uuid, name , public_rsa_key) VALUES (?,?,?)`
//_, err := r.db.Exec(query, u.UUID, u.name, u.public_rsa_key)
//	return err

// ------- //

func (sv *Server) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	UUID := generateUUID()

	scanner := bufio.NewScanner(conn)

	var (
		userIn string
		passIn string
		keyIn  string
	)

	if scanner.Scan() {
		userIn = scanner.Text()
	}
	if scanner.Scan() {
		passIn = scanner.Text()
	}
	if scanner.Scan() {
		keyIn = scanner.Text()
	}

	User := storage.Users{
		UUID:           UUID,
		Username:       userIn,
		Password_hash:  passIn,
		Public_rsa_key: keyIn,
	}
	// DEBUG HERE
	sv.mu.Lock()
	sv.clients[UUID] = conn
	sv.mu.Unlock()

	err := sv.SQLiteRepository.CreateUser(User)
	if err != nil {
		fmt.Printf("[!] Error saving to DB: %v\n", err)
		return
	}

	defer func() {
		sv.mu.Lock()
		delete(sv.clients, UUID)
		sv.mu.Unlock()
		conn.Close()
		fmt.Printf("[-] Connection closed: %s (ID: %s)\n", remoteAddr, UUID)
	}()

	for scanner.Scan() {
		msg := scanner.Text()
		formattedMsg := fmt.Sprintf("[%s]: %s", userIn, msg)
		// In production, we should avoid logging the message content for privacy
		fmt.Printf("[LOG] From (ID: %s): %s\n", UUID, formattedMsg)
		sv.broadcast(formattedMsg, conn)
	}
}

// ------- //

func (sv *Server) broadcast(msg string, sender net.Conn) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	for id, clientConn := range sv.clients {
		if clientConn != sender {
			_, err := fmt.Fprintln(clientConn, msg)
			if err != nil {
				log.Printf("[!] Could not send message to client %s: %v\n", id, err)
			}
		}
	}
}

// ------- //

func generateUUID() string {
	return uuid.New().String()
}
