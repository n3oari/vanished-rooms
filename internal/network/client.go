package network

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/ui"
)

var (
	CurrentRoom struct {
		AESKey []byte
	}
	MyPrivateKey interface{}
)

func StartClient(addr, user, pass, publicKey string) {
	ui.PrintRandomBanner()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("No se pudo conectar al servidor: %v", err)
	}
	defer conn.Close()

	fmt.Fprintln(conn, user)
	fmt.Fprintln(conn, pass)
	fmt.Fprintln(conn, publicKey)

	fmt.Printf("[+] Connected to server as %s. Say something :)\n", user)

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "KEY_DELIVERY:") {
				handleKeyDelivery(line)
				continue
			}

			if strings.HasPrefix(line, "USER_JOINED:") {
				HandleUserJoined(line, conn)
				continue
			}

			if strings.Contains(line, ": ") && !strings.HasPrefix(line, "[Server]") {
				parts := strings.SplitN(line, ": ", 2)
				usuario, junkData := parts[0], parts[1]

				if len(CurrentRoom.AESKey) > 0 {
					texto, err := cryptoutils.DecryptForChat(junkData, CurrentRoom.AESKey)
					if err == nil {
						fmt.Printf("\r[%s]: %s\n> ", usuario, texto)
						continue
					}
				}
			}

			fmt.Printf("\r%s\n> ", line)
		}
	}()

	inputScanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for inputScanner.Scan() {
		text := strings.TrimSpace(inputScanner.Text())
		if text == "" {
			continue
		}

		if strings.HasPrefix(text, "/") {
			fmt.Fprintln(conn, text)
		} else if len(CurrentRoom.AESKey) > 0 {
			basura, err := cryptoutils.EncryptForChat(text, CurrentRoom.AESKey)
			if err == nil {
				fmt.Fprintln(conn, basura)
			} else {
				fmt.Fprintln(conn, text)
			}
		} else {
			fmt.Fprintln(conn, text)
		}
		fmt.Print("> ")
	}
}

func handleKeyDelivery(line string) {
	parts := strings.Split(line, ":")
	if len(parts) < 3 {
		return
	}
	key, _ := base64.StdEncoding.DecodeString(parts[2])
	CurrentRoom.AESKey = key
	fmt.Println("\n[!] Llave recibida.")
}

func HandleUserJoined(line string, conn net.Conn) {
	fmt.Println("\n[!] Alguien entró a la sala.")
}
