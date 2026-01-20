package network

import (
	"bufio"
	"crypto/rsa"
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
	IsHost       bool
)

func StartClient(addr, user, pass, privateKeyPath string) {
	ui.PrintRandomBanner()

	if len(pass) < 10 || len(pass) > 30 {
		fmt.Println("[!] The password must be at least 10 characters long and less than 30.")
		return
	}

	privRSA, ok := MyPrivateKey.(*rsa.PrivateKey)
	if !ok {
		log.Fatal("[!] MyPrivateKey no es una llave RSA válida")
	}
	pubKeyBytes, err := cryptoutils.EncodePublicKeyToBase64(privRSA) // Asegúrate de tener esta función
	if err != nil {
		log.Fatal("Error exportando llave pública")
	}

	fmt.Println("[+] Llave privada cargada en la interfaz correctamente.")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("No se pudo conectar al servidor: %v", err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)

	fmt.Fprintln(writer, user)
	fmt.Fprintln(writer, pass)
	fmt.Fprintln(writer, pubKeyBytes)

	writer.Flush()

	fmt.Printf("[+] Connected to server as %s. Say something :)\n", user)

	go func() {
		//	scanner := bufio.NewScanner(conn)
		reader := bufio.NewReader(conn)

		for {
			//	line := scanner.Text()
			line, err := reader.ReadString('\n')
			//fmt.Printf("\n[SERVIDOR DICE]: %s\n> ", line)
			if strings.Contains(line, "created successfully") {
				newKey, err := cryptoutils.GenerateAESKey()
				if err != nil {
					fmt.Printf("\n[!] Error local al generar clave: %v\n", err)
				} else {
					CurrentRoom.AESKey = newKey
					IsHost = true
					fmt.Println("\n[!] Sala lista. Clave AES generada y guardada en memoria local.")
				}
			}

			if strings.HasPrefix(line, "KEY_DELIVERY:") {
				fmt.Printf("\n[DEBUG CLIENTE] Entrada cruda: %q\n", line)
				handleKeyDelivery(line)
				continue
			}

			if strings.HasPrefix(line, "USER_JOINED:") {
				HandleUserJoined(line, writer)
				continue
			}

			if strings.TrimSpace(line) == CmdPromotedHost {
				IsHost = true
				fmt.Println("\n[!] EL HOST ANTERIOR SE HA IDO. AHORA ERES EL DUEÑO DE LA SALA.")
				fmt.Print("> ")

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

			if line == `/quit\n` || line == `\quit\r\n` {
				fmt.Println("\n[-] ..........Disconnecting ")
				return
			}

			//			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("\n[!] El servidor ha cerrado la conexión.")
				os.Exit(0) //
			}

			fmt.Printf("\r%s ", line)
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
			fmt.Fprintln(writer, text)
		} else if len(CurrentRoom.AESKey) > 0 {
			junk, err := cryptoutils.EncryptForChat(text, CurrentRoom.AESKey)
			if err == nil {
				fmt.Printf("\n[DEBUG CLIENTE] Texto original: %s", text)
				fmt.Printf("\n[DEBUG CLIENTE] Enviando 'junk data' (AES): %s", junk)
				fmt.Printf("\n[DEBUG CLIENTE] Longitud del payload: %d bytes\n", len(junk))
				fmt.Fprintln(writer, junk)
			} else {
				fmt.Fprintln(writer, text)
			}
		} else {
			fmt.Fprintln(writer, text)
		}
		writer.Flush()
		fmt.Print("> ")
	}
}

func handleKeyDelivery(line string) {
	line = strings.TrimSpace(line)
	parts := strings.Split(line, ":")
	if len(parts) < 3 {
		return
	}
	encryptedKeyB64 := parts[2]
	fmt.Printf("\n[DEBUG CLIENTE] Recibido Payload RSA de la sala: %s...\n", encryptedKeyB64[:50])

	priv, ok := MyPrivateKey.(*rsa.PrivateKey)
	if !ok {
		fmt.Println("\n[!] Error: No se encontró la clave privada local para descifrar.")
		return
	}

	key, err := cryptoutils.DecryoptWithPrivateKey(encryptedKeyB64, priv)
	if err != nil {
		fmt.Printf("[!] Error al descifrar la llave de la sala: %v\n", err)
		return
	}

	CurrentRoom.AESKey = key
	fmt.Println("\n[!] Llave de sala recibida y activada. Ahora puedes leer los mensajes.")

}
func HandleUserJoined(line string, writer *bufio.Writer) {
	if !IsHost {
		return
	}
	parts := strings.SplitN(line, ":", 3)
	if len(parts) < 3 {
		return
	}
	targetUser := parts[1]
	pubKeyb64 := parts[2]
	fmt.Printf("[DEBUG ] RSA Pública del usuario: %s\n", targetUser, pubKeyb64)

	if len(CurrentRoom.AESKey) > 0 {
		// Ciframos nuestra AES con la RSA Pública del que acaba de entrar
		encryptedKey, err := cryptoutils.EncryptWithPublicKey(CurrentRoom.AESKey, pubKeyb64)
		if err != nil {
			fmt.Printf("\n[!] Error cifrando llave para %s: %v\n", targetUser, err)
			return
		}

		encKeyB64 := base64.StdEncoding.EncodeToString(encryptedKey)

		fmt.Fprintf(writer, "/sendKey %s %s\n", targetUser, encKeyB64)

		writer.Flush()

		fmt.Printf("\n[+] Llave de sala enviada a %s correctamente.\n> ", targetUser)
	}
}
