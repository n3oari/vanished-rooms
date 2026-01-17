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

	// when the connection ends, clean up all
	defer func() {
		sv.mu.Lock()
		roomToDelete := User.CurrentRoomUUID

		delete(sv.clients, UUID)
		delete(sv.usersInRoom, UUID)
		sv.mu.Unlock()

		// remove room
		if roomToDelete != "" {
			err := sv.SQLiteRepository.LeaveRoomAndDeleteRoomIfEmpty(UUID, roomToDelete)
			if err != nil {
				fmt.Printf("[!] Error cleaning room on disconnect: %v\n", err)
			}
		}

		// remove user
		if User.UUID != "" {
			err := sv.SQLiteRepository.DeleteUser(User)

			if err != nil {
				fmt.Printf("[!] Error deleting from DB: %v\n", err)
			} else {
				fmt.Printf("[-] User deleted from DB: %s\n", User.Username)
			}

		}

		fmt.Printf("[-] Connection closed: %s (ID: %s)\n", remoteAddr, UUID)
		conn.Close()

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
	sv.usersInRoom[UUID] = &User
	sv.mu.Unlock()

	err := sv.SQLiteRepository.CreateUser(User)
	if err != nil {
		fmt.Printf("[!] Error saving to DB: %v\n", err)
		return
	}

	for scanner.Scan() {
		//	fmt.Printf("[+] User %s connected\n", User.Username)
		msg := strings.TrimSpace(scanner.Text())
		roomMsg := fmt.Sprintf("[%s]: %s", User.Username, msg)
		if msg == "" {
			continue
		}

		if strings.HasPrefix(msg, "/") {
			handleInternalCommand(sv, conn, &User, msg)
			continue
		}

		if User.CurrentRoomUUID == "" {
			fmt.Fprintln(conn, "[!] You need to be in a room to send messages. Use /join\n")
			continue
		}

		// In production, we should avoid logging the message content for privacy
		fmt.Printf("[LOG][%s][%s]: %s\n", UUID, userIn, msg)
		sv.broadcast(roomMsg, conn, User.CurrentRoomUUID, sv.usersInRoom)
	}
}
