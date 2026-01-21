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

type InternalEvent struct {
	Type    string
	Payload string
}

type VanishedClient struct {
	conn       net.Conn
	reader     *bufio.Reader
	aesKey     []byte
	privateKey *rsa.PrivateKey
	isHost     bool
	username   string
}

func StartClient(addr string, user string, pass string, privateKeyPath string, tor bool, proxy string, privRSA *rsa.PrivateKey) {
	ui.PrintRandomBanner()

	if privRSA == nil {
		log.Fatal("[!] Private RSA Key is nil")
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("[!] Connection failed: %v", err)
	}
	defer conn.Close()

	client := &VanishedClient{
		conn:       conn,
		reader:     bufio.NewReader(conn),
		privateKey: privRSA,
		username:   user,
		isHost:     false,
		aesKey:     make([]byte, 0),
	}

	pubKeyBytes, _ := cryptoutils.EncodePublicKeyToBase64(privRSA)

	fmt.Fprintln(conn, user)
	fmt.Fprintln(conn, pass)
	fmt.Fprintln(conn, pubKeyBytes)

	go client.Listen()

	fmt.Printf("[+] Connected to server as %s.\n", user)

	inputScanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for inputScanner.Scan() {
		text := strings.TrimSpace(inputScanner.Text())
		if text == "/quit" {
			return
		}
		client.SendMessage(text)
		fmt.Print("> ")
	}
}

func (c *VanishedClient) Listen() {
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			fmt.Println("\n[!] Connection lost.")
			os.Exit(0)
		}
		event := c.parseRawLine(line)
		c.dispatch(event)
	}
}

func (c *VanishedClient) SendMessage(text string) {
	if text == "" {
		return
	}
	if strings.HasPrefix(text, "/") {
		fmt.Fprintln(c.conn, text)
	} else if len(c.aesKey) > 0 {
		encrypted, err := cryptoutils.EncryptForChat(text, c.aesKey)
		if err == nil {
			fmt.Fprintln(c.conn, encrypted)
		} else {
			fmt.Fprintln(c.conn, text)
		}
	} else {
		fmt.Fprintln(c.conn, text)
	}
}

func (c *VanishedClient) parseRawLine(line string) InternalEvent {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, EvKeyDelivery+":") {
		return InternalEvent{Type: EvKeyDelivery, Payload: line}
	}
	if strings.HasPrefix(line, EvUserJoined+":") {
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
		fmt.Println("\n[!] SYSTEM: You are now the HOST.")
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

	subCommand := parts[0]
	senderName := parts[1]
	keyData := strings.TrimSpace(parts[2])

	if subCommand == "FROM" {
		fmt.Printf("\n[DEBUG CLIENT] WRAPPED AES RECEIVED FROM %s:\n%s\n", senderName, keyData)

		decryptedKey, err := cryptoutils.DecryoptWithPrivateKey(keyData, c.privateKey)
		if err != nil {
			fmt.Printf("\n[!] Key decryption failed: %v\n", err)
			return
		}

		c.aesKey = decryptedKey
		fmt.Printf("\n[+] AES Key established. Encryption ACTIVE.\n")
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
	if len(parts) < 3 {
		return
	}

	targetUser := parts[1]
	targetPubKey := strings.TrimSpace(parts[2])

	c.processKeyRequest(targetUser, targetPubKey)
}

func (c *VanishedClient) handleSystemInfo(payload string) {
	if strings.Contains(payload, StatusRoomCreated) {
		newKey, err := cryptoutils.GenerateAESKey()
		if err == nil {
			c.aesKey = newKey
			c.isHost = true
			fmt.Printf("\n[!] SYSTEM: AES key generated. You are Host.")
		}
	}
}

func (c *VanishedClient) processKeyRequest(targetUser string, targetPubKey string) {
	if len(c.aesKey) == 0 {
		fmt.Println("\n[!] Error: No AES key to share.")
		return
	}

	encryptedBytes, err := cryptoutils.EncryptWithPublicKey(c.aesKey, targetPubKey)
	if err != nil {
		fmt.Printf("\n[!] Encryption error: %v\n", err)
		return
	}

	encKeyB64 := base64.StdEncoding.EncodeToString(encryptedBytes)

	fmt.Fprintf(c.conn, "/sendKey %s %s\n", targetUser, encKeyB64)
	fmt.Printf("\n[DEBUG HOST] Key sent to %s (Base64 encoded)\n", targetUser)
}
