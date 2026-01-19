package network

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/storage"
)

func (sv *Server) HandleConnection(conn net.Conn) {
	UUID := generateUUID()
	var User storage.Users

	defer func() {
		sv.mu.Lock()
		roomToDelete := User.CurrentRoomUUID
		delete(sv.clients, UUID)
		delete(sv.usersInRoom, UUID)
		sv.mu.Unlock()

		if roomToDelete != "" {
			sv.SQLiteRepository.LeaveRoomAndDeleteRoomIfEmpty(UUID, roomToDelete)
		}
		if User.UUID != "" {
			sv.SQLiteRepository.DeleteUser(User)
		}
		conn.Close()
	}()

	scanner := bufio.NewScanner(conn)

	if scanner.Scan() {
		User.Username = scanner.Text()
	}
	if scanner.Scan() {
		User.PasswordHash = scanner.Text()
		hash, err := cryptoutils.HashPassword(User.PasswordHash)
		if err != nil {
			fmt.Fprintf(conn, "[!] Error hashing : %v\n", err)
			return
		}
		if cryptoutils.VerifyPassword(User.PasswordHash, hash) {
			fmt.Println("[+] Hash verified successfully")
			User.PasswordHash = hash
		}

	}
	if scanner.Scan() {
		User.PublicRSAKey = scanner.Text()
	}
	User.UUID = UUID

	sv.mu.Lock()
	sv.clients[UUID] = conn
	sv.usersInRoom[UUID] = &User
	sv.mu.Unlock()
	sv.SQLiteRepository.CreateUser(User)

	fmt.Printf("[+] Nuevo usuario: %s\n", User.Username)

	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}

		sv.mu.Lock()
		u := sv.usersInRoom[UUID]
		sv.mu.Unlock()

		if strings.HasPrefix(msg, "/") {
			sv.HandleInternalCommand(conn, u, msg)
			continue
		}

		if u.CurrentRoomUUID == "" {
			fmt.Fprintln(conn, "[!] Debes entrar a una sala con /join")
			continue
		}

		fmt.Printf("[Room -> %s][%s]: %s\n", u.Username, u.Username, msg)
		// encrytped msg
		roomMsg := fmt.Sprintf("[%s]: %s", u.Username, msg)
		sv.broadcast(roomMsg, conn, u.CurrentRoomUUID, sv.usersInRoom)
	}
}
