package network

import (
	"fmt"
	"log"
	"net"
	"strings"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/storage"
)

func (sv *Server) HandleInternalCommand(conn net.Conn, user *storage.Users, msg string) {
	parts := strings.Fields(msg)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]

	switch cmd {
	case "/create":
		sv.handleCreateCommand(conn, user, msg)

	case "/join":
		sv.handleJoinCommand(conn, user, msg)

	case "/users":
		sv.handleUsersCommand(conn, user)

	case "/rooms":
		sv.handleRoomsCommand(conn)

	case "/leave-room":
		sv.handleLeaveRoomCommand(conn, user)

	case "/help":
		sv.handleHelpCommand(conn)

	case "/quit":
		sv.handleQuitCommand(conn)

	case "/sendKey":
		sv.handleSendKeyCommand(user, msg)

	default:
		fmt.Fprintf(conn, "%s:[!] Unknown command. Type /help\n", EvSystemInfo)
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

func (sv *Server) handleCreateCommand(conn net.Conn, user *storage.Users, msg string) {
	roomName := strings.TrimSpace(extractFlag(msg, "-n"))
	roomPass := strings.TrimSpace(extractFlag(msg, "-p"))
	isPrivate := strings.TrimSpace(extractFlag(msg, "--private"))

	if roomName == "" || roomPass == "" || isPrivate == "" {
		fmt.Fprintf(conn, "%s:[!] Usage: /create -n <room_name> -p <room_password> --private <y/n>\n", EvSystemInfo)
		return
	}

	privacyChoice := false
	if isPrivate == "y" {
		privacyChoice = true
	} else if isPrivate != "n" {
		fmt.Fprintf(conn, "%s:[!] Invalid privacy option. Use 'y' or 'n'.\n", EvSystemInfo)
		return
	}

	salt, err := cryptoutils.GenerarSalt()
	if err != nil {
		fmt.Fprintf(conn, "%s:[!] Error generating salt: %v\n", EvSystemInfo, err)
		return
	}

	hash := cryptoutils.HashPassword(roomPass, salt)

	newRoom := storage.Rooms{
		UUID:         generateUUID(),
		Name:         roomName,
		PasswordHash: hash,
		Salt:         salt,
		Private:      privacyChoice,
	}

	err = sv.SQLiteRepository.CreateAndJoinRoom(newRoom, user.UUID)
	if err != nil {
		fmt.Fprintf(conn, "%s:[!] Database Error: %v\n", EvSystemInfo, err)
		return
	}

	user.CurrentRoomUUID = newRoom.UUID
	user.IsOwner = true

	sv.mu.Lock()
	sv.Clients[user.UUID] = &ClientSession{
		Conn:      conn,
		ID:        user.UUID,
		Username:  user.Username,
		PublicKey: user.PublicRSAKey,
		Room:      newRoom.UUID,
	}
	sv.mu.Unlock()

	fmt.Fprintf(conn, "%s:[+] Room '%s' created. You are the host (AES generator).\n", EvSystemInfo, roomName)
}

func (sv *Server) handleJoinCommand(conn net.Conn, User *storage.Users, msg string) {

	roomName := extractFlag(msg, "-n")
	roomPass := extractFlag(msg, "-p")
	if roomName == "" || roomPass == "" {
		fmt.Fprint(conn, "[!] Usage: /join -n <room_name> -p <room_password>\n")
		return
	}

	roomID, hostID, err := sv.SQLiteRepository.JoinRoom(User.UUID, roomName, roomPass)
	if err != nil {
		fmt.Fprintf(conn, "[!] Failed to join room: %v\n", err)
		return
	}

	User.CurrentRoomUUID = roomID

	sv.mu.Lock()
	sv.Clients[User.UUID] = &ClientSession{
		Conn:      conn,
		ID:        User.UUID,
		Username:  User.Username,
		PublicKey: User.PublicRSAKey,
		Room:      roomID,
	}
	hostSession, hostExists := sv.Clients[hostID]

	sv.mu.Unlock()
	if hostExists && hostID != User.UUID {
		// Formato: KEY_DELIVERY:REQ_FROM:Username:PubKey
		fmt.Fprintf(hostSession.Conn, "%s:REQ_FROM:%s:%s\n", EvKeyDelivery, User.Username, User.PublicRSAKey)
	}

	joinNotify := fmt.Sprintf("%s:%s has joined the room", EvUserJoined, User.Username)
	sv.Broadcast(joinNotify, conn, roomID)

	fmt.Fprintf(conn, "%s:[+] Joined room: %s\n", EvSystemInfo, roomName)

}

func (sv *Server) handleUsersCommand(conn net.Conn, user *storage.Users) {
	if user.CurrentRoomUUID == "" {
		fmt.Fprintf(conn, "%s:[!] You need to be in a room. Use /join\n", EvSystemInfo)
		return
	}

	users, err := sv.SQLiteRepository.ListAllUsersInRoom(user.CurrentRoomUUID)
	if err != nil {
		fmt.Fprintf(conn, "%s:[!] Error retrieving user list\n", EvSystemInfo)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s:\n=== USER LIST ===\n", EvSystemInfo))
	for _, u := range users {
		fmt.Fprintf(&sb, " • %s\n", u.Username)
	}
	sb.WriteString("======================\n")
	conn.Write([]byte(sb.String()))
}

func (sv *Server) handleRoomsCommand(conn net.Conn) {
	rooms, err := sv.SQLiteRepository.ListAllRooms()
	if err != nil {
		log.Printf("Error al listar salas: %v", err)
		fmt.Fprintf(conn, "%s:[!] Error al recuperar las salas\n", EvSystemInfo)
		return
	}

	if len(rooms) == 0 {
		fmt.Fprintf(conn, "%s:[i] No hay salas disponibles.\n", EvSystemInfo)
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s:\n=== LISTA DE SALAS ===\n", EvSystemInfo))
	for _, room := range rooms {
		fmt.Fprintf(&sb, " • %s\n", room.Name)
	}
	sb.WriteString("======================\n")
	conn.Write([]byte(sb.String()))
}

func (sv *Server) handleLeaveRoomCommand(conn net.Conn, user *storage.Users) {
	if user.CurrentRoomUUID == "" {
		fmt.Fprintf(conn, "%s:[!] You are not in any room.\n", EvSystemInfo)
		return
	}

	roomID := user.CurrentRoomUUID
	wasOwner := user.IsOwner

	if wasOwner {
		newHostUUID, err := sv.SQLiteRepository.PromoteNextHost(roomID, user.UUID)
		if err == nil && newHostUUID != "" {
			NotifyPromotion(sv, newHostUUID)
		}
	}

	err := sv.SQLiteRepository.LeaveRoomAndDeleteRoomIfEmpty(user.UUID, roomID)
	if err != nil {
		log.Printf("[!] Error leaving the room: %v", err)
		fmt.Fprintf(conn, "%s:[!] Error leaving the room\n", EvSystemInfo)
		return
	}

	user.CurrentRoomUUID = ""
	user.IsOwner = false

	sv.mu.Lock()
	if session, exists := sv.Clients[user.UUID]; exists {
		session.Room = ""
	}
	sv.mu.Unlock()

	fmt.Fprintf(conn, "%s:[+] You left the room successfully\n", EvSystemInfo)
}

func (sv *Server) handleSendKeyCommand(sender *storage.Users, msg string) {
	parts := strings.SplitN(msg, " ", 3)
	if len(parts) < 3 {
		log.Printf("[DEBUG SERVER] Invalid /sendKey format from %s", sender.Username)
		return
	}

	targetName := parts[1]
	encryptedKey := parts[2]

	var targetConn net.Conn

	// Thread-safe search for the recipient's connection
	sv.mu.RLock()
	for _, session := range sv.Clients {
		if session.Username == targetName {
			targetConn = session.Conn
			break
		}
	}
	sv.mu.RUnlock()

	if targetConn != nil {
		fmt.Fprintf(targetConn, "%s:FROM:%s:%s\n", EvKeyDelivery, sender.Username, encryptedKey)

		log.Printf("[DEBUG SERVER] RELAYING AES KEY: [%s] -> [%s]", sender.Username, targetName)

		if len(encryptedKey) > 40 {
			log.Printf("[DEBUG SERVER] RSA-Encrypted Payload: %s...", encryptedKey[:40])
		} else {
			log.Printf("[DEBUG SERVER] RSA-Encrypted Payload: %s", encryptedKey)
		}
	} else {
		log.Printf("[DEBUG SERVER] Failed to relay key: Target user '%s' not found or offline", targetName)
	}
}

func (sv *Server) handleHelpCommand(conn net.Conn) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s:[?] Available commands:\n", EvSystemInfo)
	fmt.Fprintln(&sb, "    /create -n <name> -p <pass> --private <y/n> -> Create a new room and join")
	fmt.Fprintln(&sb, "    /rooms                        -> List all rooms")
	fmt.Fprintln(&sb, "    /join -n <name> -p <pass>    -> Join a room")
	fmt.Fprintln(&sb, "    /leave-room                  -> Leave the room")
	fmt.Fprintln(&sb, "    /users                        -> List all users (you need to be in a room)")
	fmt.Fprintln(&sb, "    /help                        -> Show help menu")
	fmt.Fprintln(&sb, "    /quit                        -> Disconnect and remove user permanently")
	fmt.Fprintln(&sb, "\n    * Tip: You can also use Control + C to quit.")
	fmt.Fprintln(&sb, "    * Generate RSA: openssl genrsa -out privada.pem 2048")

	conn.Write([]byte(sb.String()))
}

func (sv *Server) handleQuitCommand(conn net.Conn) {
	fmt.Fprintf(conn, "%s:BYE\n", EvSystemInfo)
	conn.Close()
}
