package network

import (
	"fmt"
	"log"
	"net"
)

func (sv *Server) broadcast(msg string, sender net.Conn) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	for id, clientConn := range sv.clients {
		if clientConn != sender {
			_, err := fmt.Fprintln(clientConn, msg)
			if err != nil {
				log.Printf("[!] Could not send message to client %s: %v\n", id, err)
			}
		}
	}
}
