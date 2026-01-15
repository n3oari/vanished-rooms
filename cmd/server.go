package cmd

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/google/uuid"
)

type Server struct {
	clients map[net.Conn]bool // permitira hacer Broadcast
	mu      sync.Mutex        //
}

// ------ //

func StartServer(port string) {

	sv := &Server{
		clients: make(map[net.Conn]bool),
	}
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Println(err)
		return
	}
	defer listener.Close()

	log.Printf("[+] Vanished Rooms servers listening in port: %s...\n", port)

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println(err)
			continue
		}
		go sv.handleConnection(conn)
	}

}

// ------- //

func generateUUID() string {
	return uuid.New().String()

}

// ------- //

func (s *Server) handleConnection(conn net.Conn) {

	scanner := bufio.NewScanner(conn)
	var username string

	if scanner.Scan() {
		username = scanner.Text()
		fmt.Printf("[+] User %s connected with ID %s\n", username, generateUUID())
	}

	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	// cuando la funcion termine, es decir, el cliente se desconecte, eliminamos al cliente y su conexion
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
		fmt.Printf("[-] Connection closed: %s\n", conn.RemoteAddr().String())
	}()

	for scanner.Scan() {
		msg := scanner.Text()
		fmt.Printf("[+] Message recieved from client -> %s\n", msg)
		s.broadcast(msg, conn)
	}

}

// ------ //

func (s *Server) broadcast(msg string, sender net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for client := range s.clients {
		if client != sender {
			fmt.Fprintln(client, msg)
		}
	}
}

// ------ //
