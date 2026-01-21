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
	var user storage.Users

	defer func() {
		sv.mu.Lock()
		session, exists := sv.Clients[UUID]
		if !exists {
			sv.mu.Unlock()
			return
		}

		roomID := session.Room
		userName := session.Username

		delete(sv.Clients, UUID)
		sv.mu.Unlock()

		if roomID != "" {
			log.Printf("[DEBUG] User %s exiting. Cleaning up room...", userName)
			newHost, err := sv.SQLiteRepository.PromoteNextHost(roomID, UUID)
			if err == nil && newHost != "" {
				log.Printf("[DEBUG] New host found for room %s: %s", roomID, newHost)
				NotifyPromotion(sv, newHost)
			}
			sv.SQLiteRepository.LeaveRoomAndDeleteRoomIfEmpty(UUID, roomID)
		}

		sv.SQLiteRepository.DeleteUser(storage.Users{UUID: UUID})
		conn.Close()
		log.Printf("[DEBUG] Connection closed for UUID: %s", UUID)
	}()

	scanner := bufio.NewScanner(conn)

	if !scanner.Scan() {
		return
	}
	user.Username = scanner.Text()

	if !scanner.Scan() {
		return
	}
	plainPassword := scanner.Text()

	if !scanner.Scan() {
		return
	}
	user.PublicRSAKey = scanner.Text()

	salt, _ := cryptoutils.GenerarSalt()
	user.PasswordHash = cryptoutils.HashPassword(plainPassword, salt)
	user.Salt = salt
	user.UUID = UUID

	sv.SQLiteRepository.CreateUser(user)

	sv.mu.Lock()
	sv.Clients[UUID] = &ClientSession{
		Conn:      conn,
		ID:        UUID,
		Username:  user.Username,
		PublicKey: user.PublicRSAKey,
		Room:      "",
	}
	sv.mu.Unlock()

	log.Printf("[DEBUG] New authenticated user: %s (ID: %s)", user.Username, UUID)

	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}

		sv.mu.RLock()
		session, exists := sv.Clients[UUID]
		sv.mu.RUnlock()

		if !exists {
			break
		}

		if strings.HasPrefix(msg, "/") {
			sv.HandleInternalCommand(conn, &user, msg)
			continue
		}

		if session.Room == "" {
			fmt.Fprintf(conn, "%s:[!] You must join a room first using /join\n", EvSystemInfo)
			continue
		}

		log.Printf("[DEBUG SERVER] INCOMING JUNK DATA from %s: %s", user.Username, msg)

		roomMsg := fmt.Sprintf("%s:[%s]: %s", EvChatMsg, user.Username, msg)
		sv.Broadcast(roomMsg, conn, session.Room)
	}
}

func NotifyPromotion(sv *Server, newHostUUID string) {
	sv.mu.RLock()
	session, exists := sv.Clients[newHostUUID]
	sv.mu.RUnlock()

	if !exists {
		log.Printf("[DEBUG] Could not find session for new host UUID: %s", newHostUUID)
		return
	}

	fmt.Fprintf(session.Conn, "%s:PROMOTED\n", EvHostPromoted)

	msg := fmt.Sprintf("%s:[!] User %s has been promoted to Host.", EvSystemInfo, session.Username)
	sv.Broadcast(msg, nil, session.Room)
	log.Printf("[DEBUG] Host promotion notification sent to %s", session.Username)
}
