package network

import (
	"fmt"
	"strings"
	"time"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/logger"
	"vanished-rooms/internal/storage"

	"github.com/gorilla/websocket"
)

func (sv *Server) HandleConnection(ws *websocket.Conn) {
	UUID := generateUUID()
	var user storage.Users

	defer func() {
		sv.mu.Lock()
		session, exists := sv.Clients[UUID]

		if !exists {
			sv.mu.Unlock()
			sv.SQLiteRepository.DeleteUser(storage.Users{UUID: UUID})
			ws.Close()
			return
		}

		roomID := session.Room
		userName := session.Username

		delete(sv.Clients, UUID)
		sv.mu.Unlock()

		if roomID != "" {
			l.Log(logger.INFO, fmt.Sprintf("User %s exiting. Cleaning up room resources...", userName))
			newHost, err := sv.SQLiteRepository.PromoteNextHost(roomID, UUID)
			if err == nil && newHost != "" {
				l.Log(logger.INFO, fmt.Sprintf("New host assigned for room %s: %s", roomID, newHost))
				NotifyPromotion(sv, newHost)
			}
			sv.SQLiteRepository.LeaveRoomAndDeleteRoomIfEmpty(UUID, roomID)
		}

		sv.SQLiteRepository.DeleteUser(storage.Users{UUID: UUID})
		ws.Close()
		l.Log(logger.INFO, "Connection closed and UUID purged: "+UUID)
	}()

	_, msgBytes, err := ws.ReadMessage()
	if err != nil {
		return
	}
	user.Username = string(msgBytes)

	_, msgBytes, err = ws.ReadMessage()
	if err != nil {
		return
	}
	plainPassword := string(msgBytes)

	_, msgBytes, err = ws.ReadMessage()
	if err != nil {
		return
	}
	user.PublicRSAKey = string(msgBytes)

	salt, _ := cryptoutils.GenerarSalt()
	user.PasswordHash = cryptoutils.HashPassword(plainPassword, salt)
	user.Salt = salt
	user.UUID = UUID

	err = sv.SQLiteRepository.CreateUser(user)
	if err != nil {
		l.Log(logger.ERROR, "Failed to create user in DB: "+err.Error())

		errMsg := fmt.Sprintf("%s:[!] Registration failed. Username might be taken.", EvSystemInfo)
		ws.WriteMessage(websocket.TextMessage, []byte(errMsg))

		return
	}

	sv.mu.Lock()
	sv.Clients[UUID] = &ClientSession{
		wsConn:    ws,
		ID:        UUID,
		Username:  user.Username,
		PublicKey: user.PublicRSAKey,
		Room:      "",
	}
	sv.mu.Unlock()

	l.Log(logger.INFO, fmt.Sprintf("New authenticated session: %s", user.Username))

	time.Sleep(100 * time.Millisecond)
	sv.sendAutoRooms(ws)

	welcomeMsg := fmt.Sprintf("%s: Use /rooms to list public rooms. Join using /join -n <name>", EvSystemInfo)
	ws.WriteMessage(websocket.TextMessage, []byte(welcomeMsg))

	for {
		_, messageBytes, err := ws.ReadMessage()
		if err != nil {
			break
		}

		msg := strings.TrimSpace(string(messageBytes))
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
			sv.HandleInternalCommand(ws, &user, msg)
			continue
		}

		if session.Room == "" {
			sysMsg := fmt.Sprintf("%s:[!] You must join a room first using /join", EvSystemInfo)
			ws.WriteMessage(websocket.TextMessage, []byte(sysMsg))
			continue
		}

		l.Log(logger.MSG, fmt.Sprintf(" %s: %s", user.Username, msg))

		roomMsg := fmt.Sprintf("%s:[%s]: %s", EvChatMsg, user.Username, msg)
		sv.Broadcast(roomMsg, ws, session.Room)
	}
}

func NotifyPromotion(sv *Server, newHostUUID string) {
	sv.mu.RLock()
	session, exists := sv.Clients[newHostUUID]
	sv.mu.RUnlock()

	if !exists {
		return
	}

	promotionMsg := fmt.Sprintf("%s:PROMOTED", EvHostPromoted)
	session.wsConn.WriteMessage(websocket.TextMessage, []byte(promotionMsg))

	sysMsg := fmt.Sprintf("%s:[!] User %s has been promoted to Host.", EvSystemInfo, session.Username)
	sv.Broadcast(sysMsg, nil, session.Room)

	l.Log(logger.INFO, "Host promotion notification sent to: "+session.Username)
}
