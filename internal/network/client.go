package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func StartClient(addr string, user string, pass string) {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("No se pudo conectar al servidor: %v", err)
	}
	defer conn.Close()

	fmt.Fprintln(conn, user)

	fmt.Printf("[+] Connected to server as %s. Say something :)\n", user)

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Println("\n[+] Message received: ", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Println("[-] Error leyendo del servidor: %v\n", err)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprintln(conn, scanner.Text())
	}
}
