package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

// StartClient recibe los datos capturados por Cobra
func StartClient(addr string, user string, pass string) {

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("No se pudo conectar al servidor: %v", err)
	}
	defer conn.Close()

	// --- EL PASO QUE FALTA: ENVIAR CREDENCIALES ---
	// Enviamos el primer mensaje con el formato que pactamos para el servidor
	fmt.Fprintf(conn, "%s,%s\n", user, pass)

	fmt.Printf("[+] Connected to server as %s. Say something :)\n", user)

	// Goroutine para recibir mensajes
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Println("\n[+] Message received: ", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("[-] Error leyendo del servidor: %v\n", err)
		}
	}()

	// Bucle principal para enviar mensajes desde teclado
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Fprintln(conn, scanner.Text())
	}
}
