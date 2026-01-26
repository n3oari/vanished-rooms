package network

import (
	"fmt"
	"log"
	"strings"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/storage"

	"github.com/gorilla/websocket"
)

func (sv *Server) HandleInternalCommand(conn *websocket.Conn, user *storage.Users, msg string) {
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
		errMsg := fmt.Sprintf("%s:[!] Unknown command. Type /help", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(errMsg))
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

func (sv *Server) handleCreateCommand(conn *websocket.Conn, user *storage.Users, msg string) {
	roomName := strings.TrimSpace(extractFlag(msg, "-n"))
	roomPass := strings.TrimSpace(extractFlag(msg, "-p"))
	isPrivate := strings.TrimSpace(extractFlag(msg, "--private"))

	if roomName == "" || roomPass == "" || isPrivate == "" {
		usage := fmt.Sprintf("%s:[!] Usage: /create -n <room_name> -p <room_password> --private <y/n>", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(usage))
		return
	}

	privacyChoice := false
	if isPrivate == "y" {
		privacyChoice = true
	} else if isPrivate != "n" {
		invalidPriv := fmt.Sprintf("%s:[!] Invalid privacy option. Use 'y' or 'n'.", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(invalidPriv))
		return
	}

	salt, err := cryptoutils.GenerarSalt()
	if err != nil {
		saltErr := fmt.Sprintf("%s:[!] Error generating salt: %v", EvSystemInfo, err)
		conn.WriteMessage(websocket.TextMessage, []byte(saltErr))
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
		dbErr := fmt.Sprintf("%s:[!] Database Error: %v", EvSystemInfo, err)
		conn.WriteMessage(websocket.TextMessage, []byte(dbErr))
		return
	}

	user.CurrentRoomUUID = newRoom.UUID
	user.IsOwner = true

	sv.mu.Lock()
	sv.Clients[user.UUID] = &ClientSession{
		wsConn:    conn,
		ID:        user.UUID,
		Username:  user.Username,
		PublicKey: user.PublicRSAKey,
		Room:      newRoom.UUID,
	}
	sv.mu.Unlock()

	success := fmt.Sprintf("%s:[+] Room '%s' created. You are the host (AES generator).", EvSystemInfo, roomName)
	conn.WriteMessage(websocket.TextMessage, []byte(success))
}

func (sv *Server) handleJoinCommand(conn *websocket.Conn, User *storage.Users, msg string) {

	roomName := extractFlag(msg, "-n")
	roomPass := extractFlag(msg, "-p")
	if roomName == "" || roomPass == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("[!] Usage: /join -n <room_name> -p <room_password>"))
		return
	}

	roomID, hostID, err := sv.SQLiteRepository.JoinRoom(User.UUID, roomName, roomPass)
	if err != nil {
		failMsg := fmt.Sprintf("[!] Failed to join room: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(failMsg))
		return
	}

	User.CurrentRoomUUID = roomID

	sv.mu.Lock()
	sv.Clients[User.UUID] = &ClientSession{
		wsConn:    conn,
		ID:        User.UUID,
		Username:  User.Username,
		PublicKey: User.PublicRSAKey,
		Room:      roomID,
	}
	hostSession, hostExists := sv.Clients[hostID]

	sv.mu.Unlock()
	if hostExists && hostID != User.UUID {
		// Formato: KEY_DELIVERY:REQ_FROM:Username:PubKey
		deliveryReq := fmt.Sprintf("%s:REQ_FROM:%s:%s", EvKeyDelivery, User.Username, User.PublicRSAKey)
		hostSession.wsConn.WriteMessage(websocket.TextMessage, []byte(deliveryReq))
	}

	joinNotify := fmt.Sprintf("%s:%s has joined the room", EvUserJoined, User.Username)
	sv.Broadcast(joinNotify, conn, roomID)

	successJoin := fmt.Sprintf("%s:[+] Joined room: %s", EvSystemInfo, roomName)
	conn.WriteMessage(websocket.TextMessage, []byte(successJoin))

}

func (sv *Server) handleUsersCommand(conn *websocket.Conn, user *storage.Users) {
	if user.CurrentRoomUUID == "" {
		noRoom := fmt.Sprintf("%s:[!] You need to be in a room. Use /join", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(noRoom))
		return
	}

	users, err := sv.SQLiteRepository.ListAllUsersInRoom(user.CurrentRoomUUID)
	if err != nil {
		errRet := fmt.Sprintf("%s:[!] Error retrieving user list", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(errRet))
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s:\n=== USER LIST ===\n", EvSystemInfo))
	for _, u := range users {
		fmt.Fprintf(&sb, " • %s\n", u.Username)
	}
	sb.WriteString("======================")
	conn.WriteMessage(websocket.TextMessage, []byte(sb.String()))
}

func (sv *Server) handleRoomsCommand(conn *websocket.Conn) {
	rooms, err := sv.SQLiteRepository.ListAllRooms()
	if err != nil {
		log.Printf("Error al listar salas: %v", err)
		errRooms := fmt.Sprintf("%s:[!] Error al recuperar las salas", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(errRooms))
		return
	}

	if len(rooms) == 0 {
		noRooms := fmt.Sprintf("%s:[i] No hay salas disponibles.", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(noRooms))
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s:\n=== LISTA DE SALAS ===\n", EvSystemInfo))
	for _, room := range rooms {
		fmt.Fprintf(&sb, " • %s\n", room.Name)
	}
	sb.WriteString("======================")
	conn.WriteMessage(websocket.TextMessage, []byte(sb.String()))
}

func (sv *Server) handleLeaveRoomCommand(conn *websocket.Conn, user *storage.Users) {
	if user.CurrentRoomUUID == "" {
		notInRoom := fmt.Sprintf("%s:[!] You are not in any room.", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(notInRoom))
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
		errLeave := fmt.Sprintf("%s:[!] Error leaving the room", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(errLeave))
		return
	}

	user.CurrentRoomUUID = ""
	user.IsOwner = false

	sv.mu.Lock()
	if session, exists := sv.Clients[user.UUID]; exists {
		session.Room = ""
	}
	sv.mu.Unlock()

	successLeave := fmt.Sprintf("%s:[+] You left the room successfully", EvSystemInfo)
	conn.WriteMessage(websocket.TextMessage, []byte(successLeave))
}

func (sv *Server) handleSendKeyCommand(sender *storage.Users, msg string) {
	parts := strings.SplitN(msg, " ", 3)
	if len(parts) < 3 {
		log.Printf("[DEBUG SERVER] Invalid /sendKey format from %s", sender.Username)
		return
	}

	targetName := parts[1]
	encryptedKey := parts[2]

	var targetSession *ClientSession

	// Thread-safe search for the recipient's connection
	sv.mu.RLock()
	for _, session := range sv.Clients {
		if session.Username == targetName {
			targetSession = session
			break
		}
	}
	sv.mu.RUnlock()

	if targetSession != nil {
		deliveryMsg := fmt.Sprintf("%s:FROM:%s:%s", EvKeyDelivery, sender.Username, encryptedKey)
		targetSession.wsConn.WriteMessage(websocket.TextMessage, []byte(deliveryMsg))

		log.Printf("[DEBUG SERVER] RELAYING AES KEY: [%s] -> [%s]", sender.Username, targetName)
	} else {
		log.Printf("[DEBUG SERVER] Failed to relay key: Target user '%s' not found or offline", targetName)
	}
}

func (sv *Server) handleHelpCommand(conn *websocket.Conn) {
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

	conn.WriteMessage(websocket.TextMessage, []byte(sb.String()))
}

func (sv *Server) handleQuitCommand(conn *websocket.Conn) {
	bye := fmt.Sprintf("%s:BYE", EvSystemInfo)
	conn.WriteMessage(websocket.TextMessage, []byte(bye))
	conn.Close()
}
