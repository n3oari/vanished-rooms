package network

import (
	"fmt"
	"log"
	"net"
)

func (sv *Server) Broadcast(msg string, sender net.Conn, roomUUID string) {
	if roomUUID == "" {
		return
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()

	for id, session := range sv.Clients {
		if session.Conn == sender {
			continue
		}

		if session.Room == roomUUID {
			_, err := fmt.Fprintln(session.Conn, msg)
			if err != nil {
				log.Printf("[!] Error sending message to %s (ID: %s): %v\n", session.Username, id, err)
			}
		}
	}
}
