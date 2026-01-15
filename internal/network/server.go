package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"vanished-rooms/internal/ui"

	"github.com/google/uuid"
)

type Server struct {
	clients map[string]net.Conn
	mu      sync.Mutex
}

func StartServer(port string) {
	ui.PrintBanner()
	sv := &Server{
		clients: make(map[string]net.Conn),
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	log.Printf("[+] Vanished Rooms server listening on port: %s...\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go sv.handleConnection(conn)
	}
}

func generateUUID() string {
	return uuid.New().String()
}

func (sv *Server) handleConnection(conn net.Conn) {
	// Capture client's network information
	remoteAddr := conn.RemoteAddr().String()
	clientID := generateUUID()

	scanner := bufio.NewScanner(conn)
	var username string

	if scanner.Scan() {
		username = scanner.Text()
		formatedUsername := strings.TrimSpace(username)
		fmt.Printf("[+] User '%s' connected. IP: %s | ID: %s\n", formatedUsername, remoteAddr, clientID)
	}
	// DEBUG HERE
	sv.mu.Lock()
	sv.clients[clientID] = conn
	sv.mu.Unlock()

	defer func() {
		sv.mu.Lock()
		delete(sv.clients, clientID)
		sv.mu.Unlock()
		conn.Close()
		fmt.Printf("[-] Connection closed: %s (ID: %s)\n", remoteAddr, clientID)
	}()

	for scanner.Scan() {
		msg := scanner.Text()
		formattedMsg := fmt.Sprintf("[%s]: %s", username, msg)
		// In production, we should avoid logging the message content for privacy
		fmt.Printf("[LOG] From (ID: %s): %s\n", clientID, formattedMsg)
		sv.broadcast(formattedMsg, conn)
	}
}

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
