package network

import (
	"log"

	"github.com/gorilla/websocket"
)

// Cambiamos net.Conn por *websocket.Conn
func (sv *Server) Broadcast(msg string, sender *websocket.Conn, roomUUID string) {
	if roomUUID == "" {
		return
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()

	for id, session := range sv.Clients {
		// Comparamos los punteros de los WebSockets
		if session.wsConn == sender {
			continue
		}

		if session.Room == roomUUID {
			// Usamos WriteMessage en lugar de Fprintln
			err := session.wsConn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				log.Printf("[!] Error sending message to %s (ID: %s): %v\n", session.Username, id, err)
			}
		}
	}
}
