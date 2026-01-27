package network

import (
	"log"

	"github.com/gorilla/websocket"
)

func (sv *Server) Broadcast(msg string, sender *websocket.Conn, roomUUID string) {
	if roomUUID == "" {
		return
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()

	for id, session := range sv.Clients {
		if session.wsConn == sender {
			continue
		}

		if session.Room == roomUUID {
			err := session.wsConn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Printf("[!] Error sending message to %s (ID: %s): %v\n", session.Username, id, err)
			}
		}
	}
}
