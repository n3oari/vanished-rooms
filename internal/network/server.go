package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"vanished-rooms/internal/storage"
	"vanished-rooms/internal/ui"

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

// ------- //

func (sv *Server) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	UUID := generateUUID()
	var User storage.Users

	defer func() {
		sv.mu.Lock()
		delete(sv.clients, UUID)
		sv.mu.Unlock()

		if User.UUID != "" {
			err := sv.SQLiteRepository.DeleteUser(User)
			if err != nil {
				fmt.Printf("[!] Error deleting from DB: %v\n", err)
			} else {
				fmt.Printf("[-] User deleted from DB: %s (ID: %s)\n", User.Username, UUID)
			}
		}
		conn.Close()
		fmt.Printf("[-] Connection closed: %s (ID: %s)\n", remoteAddr, UUID)
	}()

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

	User = storage.Users{
		UUID:         UUID,
		Username:     userIn,
		PasswordHash: passIn,
		PublicRSAKey: keyIn,
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

	fmt.Printf("[+] User %s connected\n", User.Username)

	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}

		if strings.HasPrefix(msg, "/") {
			handleInternalCommand(sv, conn, &User, msg)
			continue
		}
		// In production, we should avoid logging the message content for privacy
		fmt.Printf("[LOG] From (ID: %s): %s\n", UUID, msg)
		sv.broadcast(msg, conn)
	}
}

func handleInternalCommand(sv *Server, conn net.Conn, User *storage.Users, msg string) {
	parts := strings.Fields(msg)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "/create":
		roomName := extractFlag(msg, "-n")
		roomPass := extractFlag(msg, "-p")

		if roomName == "" || roomPass == "" {
			fmt.Fprintln(conn, "[!] Usage: /create -n <room_name> -p <room_password>")
			return
		}

		newRoom := storage.Rooms{
			UUID:         generateUUID(),
			Name:         roomName,
			PasswordHash: roomPass,
		}

		// Database call
		err := sv.SQLiteRepository.CreateAndJoinRoom(newRoom, User.UUID)
		if err != nil {
			fmt.Fprintf(conn, "[!] Database Error: %v\n", err)
		} else {
			fmt.Fprintf(conn, "[+] Room '%s' created successfully.\n", roomName)
		}

	case "/help":
		fmt.Fprintln(conn, "[?] Available commands:")
		fmt.Fprintln(conn, "    /create -n <name> -p <pass>  -> Create a new room")
		fmt.Fprintln(conn, "    /help                        -> Show this message")
		fmt.Fprintln(conn, "    /quit                        -> Disconnect and Remove user permanetly")
	case "/quit":
		fmt.Fprintln(conn, "[!] ¡¡ BYEE !!")
		conn.Close()
		return

	default:
		fmt.Fprintf(conn, "[!] Unknown command: %s. Type /help for info.\n", parts[0])
	}
}

// idx+1 < len(parts) is for security: ensures the flag isn't the last word
// and prevents an "index out of range" panic.
func extractFlag(msg, flag string) string {
	parts := strings.Fields(msg)
	for idx, part := range parts {
		if part == flag && idx+1 < len(parts) {
			return parts[idx+1]
		}
	}
	return ""
}

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
