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
	ui.PrintRandomBanner()
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
			if User.CurrentRoomUUID != "" {
				sv.SQLiteRepository.RemoveParticipant(User.UUID, User.CurrentRoomUUID)
				deleted, err := sv.SQLiteRepository.DeleteRoomIfEmpty(User.CurrentRoomUUID)

				if err != nil {
					fmt.Printf("[!] Error cleaning room: %v\n", err)
				} else if deleted {
					fmt.Printf("[!] Vanished: Room %s was empty and has been deleted.\n", User.CurrentRoomUUID)
				} else {
					fmt.Printf("[-] Room %s still has participants.\n", User.CurrentRoomUUID)
				}
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

	for scanner.Scan() {
		//	fmt.Printf("[+] User %s connected\n", User.Username)
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}

		if strings.HasPrefix(msg, "/") {
			handleInternalCommand(sv, conn, &User, msg)
			continue
		}
		// In production, we should avoid logging the message content for privacy
		fmt.Printf("[LOG][%s][%s]: %s\n", UUID, userIn, msg)
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
	case "/join":
		roomName := extractFlag(msg, "-n")
		roomPass := extractFlag(msg, "-p")
		if roomName == "" || roomPass == "" {
			fmt.Fprintln(conn, "[!] Usage: /join -n <room_name> -p <room_password>")
			return
		}
		roomID, err := sv.SQLiteRepository.JoinRoom(User.UUID, roomName, roomPass)
		if err != nil {
			fmt.Fprintf(conn, "[!] Failed to join room: %v\n", err)
			return
		}

		User.CurrentRoomUUID = roomID

		// 3. Confirmación de éxito
		fmt.Fprintf(conn, "[+] Success! You have joined the room: %s\n", roomName)

	case "/rooms":
		rooms, err := sv.SQLiteRepository.ListAllRooms()
		if err != nil {
			conn.Write([]byte("Error retrieving rooms\n"))
			return
		}

		var sb strings.Builder
		sb.WriteString("ROOMS_LIST_START\n")
		for _, room := range rooms {
			sb.WriteString("- ")
			sb.WriteString(room.Name)
			sb.WriteString("\n")
		}
		//	sb.WriteString("EOF\n")

		_, err = conn.Write([]byte(sb.String()))
		if err != nil {
			log.Print("Error sending data to client")
		}

	case "/help":
		fmt.Fprintln(conn, "[?] Available commands:")
		fmt.Fprintln(conn, "    /rooms                       -> List all rooms")
		fmt.Fprintln(conn, "    /create -n <name> -p <pass>  -> Create a new room and join")
		fmt.Fprintln(conn, "    /join -n <name> -p <pass>    -> Join a room")
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
