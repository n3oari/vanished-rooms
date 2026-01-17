package network

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"vanished-rooms/internal/storage"
)

func (sv *Server) HandleConnection(conn net.Conn) {
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
	)

	if scanner.Scan() {
		userIn = scanner.Text()
	}
	if scanner.Scan() {
		passIn = scanner.Text()
	}

	User = storage.Users{
		UUID:         UUID,
		Username:     userIn,
		PasswordHash: passIn,
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
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}

		// 1. OBTENER SIEMPRE LA VERSIÓN REAL DEL USUARIO
		sv.mu.Lock()
		u := sv.usersInRoom[UUID]
		sv.mu.Unlock()

		// 2. PROCESAR COMANDOS
		if strings.HasPrefix(msg, "/") {
			sv.HandleInternalCommand(conn, u, msg)
			continue
		}

		// 3. VALIDAR SALA USANDO LA VERSIÓN REAL (u)
		if u.CurrentRoomUUID == "" {
			fmt.Fprintln(conn, "[!] You need to be in a room to send messages. Use /join")
			continue
		}

		// 4. BROADCAST USANDO EL ID ACTUALIZADO
		roomMsg := fmt.Sprintf("[%s]: %s", u.Username, msg)
		fmt.Printf("[LOG][%s][%s]: %s\n", UUID, u.Username, msg)
		sv.broadcast(roomMsg, conn, u.CurrentRoomUUID, sv.usersInRoom)
	}
}
