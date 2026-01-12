package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

// cambiar a servidor privado cuando este terminada la app
var addr = "localhost:8080"

func main() {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("No se pudo conectar al servidor: %v", err)
	}
	defer conn.Close()

	fmt.Println("[+] Connected to server. Say something :)")

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Println("[+] Message recieved from other user ----------------> ", scanner.Text())
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprintln(conn, scanner.Text())
	}

}
