package network

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"vanished-rooms/internal/storage"
)

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
