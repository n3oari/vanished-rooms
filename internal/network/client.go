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
	"sync"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/logger"
	"vanished-rooms/internal/ui"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
)

const (
	ServerAddrLocal = "ws://127.0.0.1:8080/ws"
	ServerAddrTor   = "ws://wuopotpej2uap77giiz7xlpw5mqjdcmpjftmnxsprp6thjib2oyunoid.onion/ws"
	TorProxyAddr    = "127.0.0.1:9050"
)

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
	writeMu    sync.Mutex
	l          *logger.CustomLogger
}

func StartClient(user string, pass string, privRSA *rsa.PrivateKey, useTor bool) {
	ui.PrintRandomBanner()
	l := logger.New()

	if privRSA == nil {
		log.Fatal("[!] Private RSA Key is nil")
	}

	var serverAddr string
	if useTor {
		serverAddr = ServerAddrTor
	} else {
		serverAddr = ServerAddrLocal
	}

	dialer := websocket.DefaultDialer
	l.Log(logger.WARN, "Connecting to the wire...")

	if useTor {
		l.Log(logger.INFO, "Tor mode enabled. Configuring SOCKS5 tunnel...")
		socksDialer, err := proxy.SOCKS5("tcp", TorProxyAddr, nil, proxy.Direct)
		if err != nil {
			log.Fatalf("[!] Error: Could not connect to Tor at %s. Is the service running?", TorProxyAddr)
		}
		dialer.NetDial = func(network, addr string) (net.Conn, error) {
			return socksDialer.Dial(network, addr)
		}
	} else {
		l.Log(logger.INFO, "Direct connection mode. Bypassing Tor proxy.")
	}

	l.Log(logger.INFO, "Establishing connection to: "+serverAddr)
	conn, _, err := dialer.Dial(serverAddr, nil)
	if err != nil {
		log.Fatalf("[!] Connection error: %v\n[?] Ensure the server is online at %s", err, serverAddr)
	}
	l.Log(logger.WARN, "Connected successfully!")
	defer conn.Close()

	client := &VanishedClient{
		wsConn:     conn,
		privateKey: privRSA,
		username:   user,
		isHost:     false,
		aesKey:     make([]byte, 0),
		l:          l,
	}

	if len(pass) < 8 {
		fmt.Println("\r[!] Security Error: Password too short.")
		fmt.Println("[i] For your safety, passwords must be at least 8 characters long.")
		client.Close("Insecure password")
		os.Exit(1)
	}

	pubKeyBytes, _ := cryptoutils.EncodePublicKeyToBase64(privRSA)

	client.Send([]byte(user))
	client.Send([]byte(pass))
	client.Send([]byte(pubKeyBytes))

	go client.Listen()

	fmt.Printf("[+] Connection successful. Welcome to the secure environment, %s.\n", user)

	inputScanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for inputScanner.Scan() {
		text := strings.TrimSpace(inputScanner.Text())
		if text == "" {
			fmt.Print("> ")
			continue
		}

		if len(text) > 1024 {
			l.Log(logger.WARN, "Message too long. Maximum 1024 characters.")
			continue
		}

		if text == "/quit" {
			client.Close("")
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
			fmt.Println("\n[!] Connection lost with server.")
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
	c.Send([]byte(finalMsg))
}

func (c *VanishedClient) Send(msg []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.wsConn.WriteMessage(websocket.TextMessage, msg)
}

func (c *VanishedClient) Close(reason string) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason))
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
		c.l.Log(logger.INFO, "SYSTEM: You have been promoted to room HOST.")
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

	subCommand, _, keyData := parts[0], parts[1], strings.TrimSpace(parts[2])

	if subCommand == "FROM" {
		decryptedKey, err := cryptoutils.DecryoptWithPrivateKey(keyData, c.privateKey)
		if err == nil {
			c.aesKey = decryptedKey
			c.l.Log(logger.INFO, "AES Key established. Chat encryption ENABLED.")
		}
	} else if subCommand == "REQ_FROM" {
		c.processKeyRequest(parts[1], keyData)
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
			c.l.Log(logger.INFO, "SYSTEM: AES Key generated. You are the room Host.")
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
		if err := c.Send([]byte(fmt.Sprintf("/sendKey %s %s", targetUser, encKeyB64))); err != nil {
			c.l.Log(logger.ERROR, fmt.Sprintf("Failed to send key: %v", err))
		}
	}
}
