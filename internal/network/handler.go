package network

import (
	"bufio"
	"fmt"
	"log"
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
		uState, exists := sv.UsersInRoom[UUID]
		if !exists {
			sv.mu.Unlock()
			return
		}

		roomID := uState.CurrentRoomUUID
		wasOwner := uState.IsOwner
		userName := uState.Username

		delete(sv.Clients, UUID)
		delete(sv.UsersInRoom, UUID)
		sv.mu.Unlock()

		if roomID != "" && wasOwner {
			log.Printf("[DEBUG] Host %s saliendo. Buscando sucesor...", userName)
			newHost, err := sv.SQLiteRepository.PromoteNextHost(roomID, UUID)
			if err == nil && newHost != "" {
				log.Printf("[DEBUG] Nuevo Host encontrado: %s", newHost)
				NotifyPromotion(sv, newHost)
			}
		}

		sv.SQLiteRepository.LeaveRoomAndDeleteRoomIfEmpty(UUID, roomID)
		sv.SQLiteRepository.DeleteUser(*uState)
		conn.Close()
	}()

	scanner := bufio.NewScanner(conn)

	if scanner.Scan() {
		User.Username = scanner.Text()
	}
	if scanner.Scan() {
		plainPassword := scanner.Text()

		salt, err := cryptoutils.GenerarSalt()
		if err != nil {
			fmt.Fprintf(conn, "[!] Error generating salt: %v\n", err)
			return
		}

		hash := cryptoutils.HashPassword(plainPassword, salt)

		if cryptoutils.VerifyPassword(plainPassword, salt, hash) {
			fmt.Println("[+] Hash verified successfully")

			User.PasswordHash = hash
			User.Salt = salt
		}

	}
	if scanner.Scan() {
		User.PublicRSAKey = scanner.Text()
	}
	User.UUID = UUID

	sv.mu.Lock()
	sv.Clients[UUID] = conn
	sv.UsersInRoom[UUID] = &User
	sv.mu.Unlock()
	sv.SQLiteRepository.CreateUser(User)

	fmt.Printf("[+] Nuevo usuario: %s\n", User.Username)

	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}

		sv.mu.Lock()
		u := sv.UsersInRoom[UUID]
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
		sv.broadcast(roomMsg, conn, u.CurrentRoomUUID, sv.UsersInRoom)
	}
}

func NotifyPromotion(sv *Server, newHostUUID string) {
	sv.mu.Lock()
	user, okUser := sv.UsersInRoom[newHostUUID]
	clientConn, okConn := sv.Clients[newHostUUID]
	sv.mu.Unlock()

	if !okUser || !okConn {
		log.Printf("[!] No se pudo encontrar al nuevo host %s en memoria para notificar", newHostUUID)
		return
	}

	_, err := fmt.Fprintln(clientConn, "SYSTEM_CMD:PROMOTED_TO_HOST")

	if err != nil {
		log.Printf("[!] Error enviando comando al nuevo host %s: %v", user.Username, err)
	} else {
		log.Printf("[+] Mensaje enviado: %s es el nuevo Host", user.Username)
	}

	msg := fmt.Sprintf("[!] El usuario %s es ahora el Host de la sala.\n", user.Username)
	sv.broadcast(msg, nil, user.CurrentRoomUUID, sv.UsersInRoom)
}
