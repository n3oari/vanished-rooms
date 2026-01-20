package network

import (
	"fmt"
	"log"
	"net"
	"vanished-rooms/internal/storage"
)

func (sv *Server) Broadcast(msg string, sender net.Conn, roomUUID string, allUsers map[string]*storage.Users) {
	if roomUUID == "" {
		return
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()

	for id, clientConn := range sv.Clients {
		if clientConn == sender {
			continue
		}
		user, exists := allUsers[id]
		if exists && user.CurrentRoomUUID == roomUUID {
			// 3. Enviamos el mensaje
			_, err := fmt.Fprintln(clientConn, msg)

			if err != nil {
				log.Printf("[!] Error sending the message a %s (ID: %s): %v\n", user.Username, id, err)
			}
		}

	}
}
