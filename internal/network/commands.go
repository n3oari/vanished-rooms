package network

import (
	"fmt"
	"strings"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/logger"
	"vanished-rooms/internal/storage"
	"vanished-rooms/internal/ui"

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
	hasPrivate := strings.Contains(msg, "--private")
	hasPublic := strings.Contains(msg, "--public")

	if (!hasPrivate && !hasPublic) || (hasPrivate && hasPublic) {
		usage := fmt.Sprintf("%s:[!] Error: Use --private or --public", EvSystemInfo)
		conn.WriteMessage(websocket.TextMessage, []byte(usage))
		return
	}

	if roomName == "" {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:[!] Error: Room name is required.", EvSystemInfo)))
		return
	}

	var hash, salt []byte
	var err error
	privacyChoice := hasPrivate

	if privacyChoice {
		if roomPass == "" {
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:[!] Error: Private rooms require a password (-p).", EvSystemInfo)))
			return
		}

		if len(roomPass) < 8 {
			conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:[!] Error: Password too short (min 8 chars).", EvSystemInfo)))
			return
		}

		salt, err = cryptoutils.GenerarSalt()
		if err != nil {
			l.Log(logger.ERROR, "Salt generation failed: "+err.Error())
			return
		}
		hash = cryptoutils.HashPassword(roomPass, salt)
	}

	newRoom := storage.Rooms{
		UUID:         generateUUID(),
		Name:         roomName,
		PasswordHash: hash,
		Salt:         salt,
		Private:      privacyChoice,
	}

	err = sv.SQLiteRepository.CreateAndJoinRoom(newRoom, user.UUID)
	if err != nil {
		l.Log(logger.ERROR, "Database error on room creation: "+err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:[!] Database Error.", EvSystemInfo)))
		return
	}

	user.CurrentRoomUUID = newRoom.UUID
	user.IsOwner = true

	sv.mu.Lock()
	if session, exists := sv.Clients[user.UUID]; exists {
		session.Room = newRoom.UUID
	}
	sv.mu.Unlock()

	l.Log(logger.INFO, fmt.Sprintf("Room created: %s by %s", roomName, user.Username))
	success := fmt.Sprintf("%s:[+] Room '%s' created. You are the host.", EvSystemInfo, roomName)
	conn.WriteMessage(websocket.TextMessage, []byte(success))
}

func (sv *Server) handleJoinCommand(conn *websocket.Conn, User *storage.Users, msg string) {
	roomName := extractFlag(msg, "-n")
	roomPass := extractFlag(msg, "-p")

	if roomName == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("[!] Usage: /join -n <name> [-p <pass>]"))
		return
	}

	roomID, hostID, err := sv.SQLiteRepository.JoinRoom(User.UUID, roomName, roomPass)
	if err != nil {
		// Mensajes específicos según el error
		var errorMsg string
		switch err.Error() {
		case "room is full":
			errorMsg = "[!] Room is full."
		case "invalid password":
			errorMsg = "[!] Access denied: Invalid password."
		case "room not found":
			errorMsg = "[!] Room not found."
		case "already in room":
			errorMsg = "[!] You are already in this room."
		default:
			errorMsg = "[!] Error joining room."
		}
		conn.WriteMessage(websocket.TextMessage, []byte(errorMsg))
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
		deliveryReq := fmt.Sprintf("%s:REQ_FROM:%s:%s", EvKeyDelivery, User.Username, User.PublicRSAKey)
		hostSession.Send([]byte(deliveryReq))
	}

	sv.Broadcast(fmt.Sprintf("%s:%s has joined the room", EvUserJoined, User.Username), conn, roomID)
	l.Log(logger.INFO, fmt.Sprintf("User %s joined room %s", User.Username, roomName))
	//	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:[+] Joined room: %s", EvSystemInfo, roomName)))

	sv.mu.Lock()
	session, exists := sv.Clients[User.UUID]
	sv.mu.Unlock()
	if exists {
		session.Send([]byte(fmt.Sprintf("%s:[+] Joined room: %s", EvSystemInfo, roomName)))
	}

}

func (sv *Server) handleUsersCommand(conn *websocket.Conn, user *storage.Users) {
	if user.CurrentRoomUUID == "" {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:[!] You are not in a room.", EvSystemInfo)))
		return
	}

	users, _ := sv.SQLiteRepository.ListAllUsersInRoom(user.CurrentRoomUUID)

	usernames := make([]string, len(users))
	for i, u := range users {
		usernames[i] = u.Username
	}

	formattedList := fmt.Sprintf("%s:\n%s", EvSystemInfo, ui.RenderUserList(usernames, user.Username))
	conn.WriteMessage(websocket.TextMessage, []byte(formattedList))
}

func (sv *Server) handleRoomsCommand(conn *websocket.Conn) {
	rooms, _ := sv.SQLiteRepository.ListAllRooms()

	roomNames := make([]string, len(rooms))
	for i, room := range rooms {
		roomNames[i] = room.Name
	}

	formattedList := fmt.Sprintf("%s:\n%s", EvSystemInfo, ui.RenderRoomList(roomNames))
	conn.WriteMessage(websocket.TextMessage, []byte(formattedList))
}

func (sv *Server) handleLeaveRoomCommand(conn *websocket.Conn, user *storage.Users) {
	if user.CurrentRoomUUID == "" {
		return
	}

	roomID := user.CurrentRoomUUID
	if user.IsOwner {
		newHost, err := sv.SQLiteRepository.PromoteNextHost(roomID, user.UUID)
		if err == nil && newHost != "" {
			NotifyPromotion(sv, newHost)
		}
	}

	sv.SQLiteRepository.LeaveRoomAndDeleteRoomIfEmpty(user.UUID, roomID)
	l.Log(logger.INFO, fmt.Sprintf("User %s left room %s", user.Username, roomID))

	user.CurrentRoomUUID = ""
	user.IsOwner = false
	sv.mu.Lock()
	if session, exists := sv.Clients[user.UUID]; exists {
		session.Room = ""
	}
	sv.mu.Unlock()

	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:[+] Left room successfully.", EvSystemInfo)))
}

func (sv *Server) handleSendKeyCommand(sender *storage.Users, msg string) {
	parts := strings.SplitN(msg, " ", 3)
	if len(parts) < 3 {
		return
	}

	targetName, encryptedKey := parts[1], parts[2]
	var targetSession *ClientSession

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
		targetSession.Send([]byte(deliveryMsg))
		l.Log(logger.ONION_INFO, fmt.Sprintf("RELAYING AES KEY: [%s] -> [%s]", sender.Username, targetName))
	}
}

func (sv *Server) handleHelpCommand(conn *websocket.Conn) {
	conn.WriteMessage(websocket.TextMessage, []byte(ui.RenderHelpMenu()))
}

func (sv *Server) handleQuitCommand(conn *websocket.Conn) {
	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%s:BYE", EvSystemInfo)))
	conn.Close()
}
