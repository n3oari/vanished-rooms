package network

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"vanished-rooms/internal/storage"
)

func (sv *Server) HandleConnection(conn net.Conn) {
	UUID := generateUUID()
	var User storage.Users

	// Limpieza al desconectar
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

	// 1. LOGIN: Recibimos los 3 datos iniciales del cliente
	if scanner.Scan() {
		User.Username = scanner.Text()
	}
	if scanner.Scan() {
		User.PasswordHash = scanner.Text()
	}
	if scanner.Scan() {
		User.PublicRSAKey = scanner.Text()
	}
	User.UUID = UUID

	// Guardar usuario en memoria y DB
	sv.mu.Lock()
	sv.clients[UUID] = conn
	sv.usersInRoom[UUID] = &User
	sv.mu.Unlock()
	sv.SQLiteRepository.CreateUser(User)

	fmt.Printf("[+] Nuevo usuario: %s\n", User.Username)

	// 2. BUCLE PRINCIPAL: Escuchar mensajes
	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}

		// Obtener datos actualizados del usuario (por si cambió de sala)
		sv.mu.Lock()
		u := sv.usersInRoom[UUID]
		sv.mu.Unlock()

		// A. Si es un comando (/join, /sendKey, etc)
		if strings.HasPrefix(msg, "/") {
			sv.HandleInternalCommand(conn, u, msg)
			continue
		}

		// B. Si es un mensaje de chat
		if u.CurrentRoomUUID == "" {
			fmt.Fprintln(conn, "[!] Debes entrar a una sala con /join")
			continue
		}

		// LOG DE SEGURIDAD: Aquí es donde verás la "Junk Data"
		// Si el cliente cifra bien, 'msg' será un Base64 loco
		fmt.Printf("[LOG][Sala: %s][%s]: %s\n", u.CurrentRoomUUID, u.Username, msg)

		roomMsg := fmt.Sprintf("[%s]: %s", u.Username, msg)
		sv.broadcast(roomMsg, conn, u.CurrentRoomUUID, sv.usersInRoom)
	}
}
