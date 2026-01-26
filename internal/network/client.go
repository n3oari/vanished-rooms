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

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const ServerOnionAddr = "ws://pxwm5oqmxvjgmfbtkyuu6izdxbov7pjwyyaejhhn7c5o4z2lv2eaxkyd.onion/ws"
const TorProxyAddr = "127.0.0.1:9050"

type InternalEvent struct {
	Type    string
	Payload string
}

type VanishedClient struct {
	wsConn     *websocket.Conn
	aesKey     []byte
	privateKey *rsa.PrivateKey
	isHost     bool
	username   string
}

func StartClient(user string, pass string, privRSA *rsa.PrivateKey) {
	ui.PrintRandomBanner()

	if privRSA == nil {
		log.Fatal("[!] Private RSA Key is nil")
	}

	socksDialer, err := proxy.SOCKS5("tcp", TorProxyAddr, nil, proxy.Direct)
	if err != nil {
		log.Fatalf("[!] Error: No se pudo contactar con Tor en %s. ¿Está el servicio activo?", TorProxyAddr)
	}

	dialer := websocket.DefaultDialer
	dialer.NetDial = func(network, addr string) (net.Conn, error) {
		return socksDialer.Dial(network, addr)
	}

	fmt.Printf("[i] Estableciendo túnel anónimo hacia: %s\n", ServerOnionAddr)

	conn, _, err := dialer.Dial(ServerOnionAddr, nil)
	if err != nil {
		log.Fatalf("[!] Error de conexión al servicio oculto: %v\n[?] Verifica que tu dirección .onion sea correcta y el servidor esté UP.", err)
	}
	defer conn.Close()

	client := &VanishedClient{
		wsConn:     conn,
		privateKey: privRSA,
		username:   user,
		isHost:     false,
		aesKey:     make([]byte, 0),
	}

	pubKeyBytes, _ := cryptoutils.EncodePublicKeyToBase64(privRSA)
	client.wsConn.WriteMessage(websocket.TextMessage, []byte(user))
	client.wsConn.WriteMessage(websocket.TextMessage, []byte(pass))
	client.wsConn.WriteMessage(websocket.TextMessage, []byte(pubKeyBytes))

	go client.Listen()

	fmt.Printf("[+] Conexión exitosa. Bienvenido al entorno seguro, %s.\n", user)

	inputScanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for inputScanner.Scan() {
		text := strings.TrimSpace(inputScanner.Text())
		if text == "" {
			fmt.Print("> ")
			continue
		}
		if text == "/quit" {
			client.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
		client.SendMessage(text)
		fmt.Print("> ")
	}
}

func (c *VanishedClient) Listen() {
	for {
		_, message, err := c.wsConn.ReadMessage()
		if err != nil {
			fmt.Println("\n[!] Conexión perdida con el servicio oculto.")
			os.Exit(0)
		}
		line := string(message)
		event := c.parseRawLine(line)
		c.dispatch(event)
	}
}

func (c *VanishedClient) SendMessage(text string) {
	var finalMsg string
	if strings.HasPrefix(text, "/") {
		finalMsg = text
	} else if len(c.aesKey) > 0 {
		encrypted, err := cryptoutils.EncryptForChat(text, c.aesKey)
		if err == nil {
			finalMsg = encrypted
		} else {
			finalMsg = text
		}
	} else {
		finalMsg = text
	}
	c.wsConn.WriteMessage(websocket.TextMessage, []byte(finalMsg))
}

func (c *VanishedClient) parseRawLine(line string) InternalEvent {
	line = strings.TrimSpace(line)
	const suffix = ":"
	if strings.HasPrefix(line, EvKeyDelivery+suffix) {
		return InternalEvent{Type: EvKeyDelivery, Payload: line}
	}
	if strings.HasPrefix(line, EvUserJoined+suffix) {
		return InternalEvent{Type: EvUserJoined, Payload: line}
	}
	if strings.Contains(line, StatusRoomCreated) {
		return InternalEvent{Type: EvSystemInfo, Payload: line}
	}
	if strings.Contains(line, "PROMOTED") {
		return InternalEvent{Type: EvHostPromoted, Payload: line}
	}
	return InternalEvent{Type: EvChatMsg, Payload: line}
}

func (c *VanishedClient) dispatch(event InternalEvent) {
	switch event.Type {
	case EvChatMsg:
		c.processIncomingChat(event.Payload)
	case EvKeyDelivery:
		c.handleKeyDelivery(event.Payload)
	case EvUserJoined:
		c.handleUserJoined(event.Payload)
	case EvSystemInfo:
		c.handleSystemInfo(event.Payload)
	case EvHostPromoted:
		c.isHost = true
		fmt.Println("\n[!] SYSTEM: Has sido promocionado a HOST de la sala.")
	}
}

func (c *VanishedClient) processIncomingChat(payload string) {
	if strings.Contains(payload, ":") {
		parts := strings.SplitN(payload, ": ", 2)
		if len(parts) == 2 {
			username, encryptedData := parts[0], strings.TrimSpace(parts[1])
			if len(c.aesKey) > 0 {
				decrypted, err := cryptoutils.DecryptForChat(encryptedData, c.aesKey)
				if err == nil {
					fmt.Printf("\r%s: %s\n> ", username, decrypted)
					return
				}
			}
		}
	}
	fmt.Printf("\r%s\n> ", payload)
}

func (c *VanishedClient) handleKeyDelivery(line string) {
	payload := strings.TrimPrefix(line, EvKeyDelivery+":")
	parts := strings.SplitN(payload, ":", 3)
	if len(parts) < 3 {
		return
	}

	subCommand, senderName, keyData := parts[0], parts[1], strings.TrimSpace(parts[2])

	if subCommand == "FROM" {
		fmt.Printf("\n[DEBUG CLIENT] WRAPPED AES RECEIVED FROM %s:\n%s\n", senderName, keyData)
		decryptedKey, err := cryptoutils.DecryoptWithPrivateKey(keyData, c.privateKey)
		if err == nil {
			c.aesKey = decryptedKey
			fmt.Printf("\n[+] Llave AES establecida. Cifrado de chat ACTIVADO.\n")
		}
	} else if subCommand == "REQ_FROM" {
		fmt.Printf("\n[DEBUG CLIENT] RSA PUBLIC KEY FROM %s:\n%s\n", senderName, keyData)
		c.processKeyRequest(senderName, keyData)
	}
}

func (c *VanishedClient) handleUserJoined(payload string) {
	if !c.isHost {
		return
	}
	parts := strings.SplitN(payload, ":", 3)
	if len(parts) >= 3 {
		c.processKeyRequest(parts[1], strings.TrimSpace(parts[2]))
	}
}

func (c *VanishedClient) handleSystemInfo(payload string) {
	if strings.Contains(payload, StatusRoomCreated) {
		newKey, err := cryptoutils.GenerateAESKey()
		if err == nil {
			c.aesKey = newKey
			c.isHost = true
			fmt.Printf("\n[!] SYSTEM: Llave AES generada. Eres el Host de la sala.")
		}
	}
}

func (c *VanishedClient) processKeyRequest(targetUser string, targetPubKey string) {
	if len(c.aesKey) == 0 {
		return
	}
	encryptedBytes, err := cryptoutils.EncryptWithPublicKey(c.aesKey, targetPubKey)
	if err == nil {
		encKeyB64 := base64.StdEncoding.EncodeToString(encryptedBytes)
		fmt.Printf("\n[DEBUG HOST] Enviando llave cifrada a %s (Base64)\n", targetUser)
		c.wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("/sendKey %s %s", targetUser, encKeyB64)))
	}
}
