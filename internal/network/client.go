package network

import (
	"bufio"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"strings"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/ui"

	"github.com/rivo/tview"
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

func StartClient(app *tview.Application, addr string, user string, pass string, privateKeyPath string, tor bool, proxy string, privRSA *rsa.PrivateKey) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("[!] Connection failed: %v", err)
	}

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

	app.QueueUpdateDraw(func() {
		ui.LaunchChatUI(app, conn, user)
	})
}

func (c *VanishedClient) Listen() {
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			ui.WriteToTerminal("SYSTEM", "yellow", "DISCONNECTED FROM NODE")
			return
		}
		event := c.parseRawLine(line)
		c.dispatch(event)
	}
}

func (c *VanishedClient) parseRawLine(line string) InternalEvent {
	line = strings.TrimSpace(line)

	if strings.Contains(line, EvKeyDelivery) {
		return InternalEvent{Type: EvKeyDelivery, Payload: line}
	}
	if strings.Contains(line, EvUserJoined) {
		return InternalEvent{Type: EvUserJoined, Payload: line}
	}
	if strings.Contains(line, EvSystemInfo) || strings.Contains(line, "LISTA DE SALAS") {
		return InternalEvent{Type: EvSystemInfo, Payload: line}
	}
	if strings.Contains(line, "PROMOTED") {
		return InternalEvent{Type: EvHostPromoted, Payload: line}
	}
	if strings.Contains(line, "CHAT_MSG") {
		return InternalEvent{Type: EvChatMsg, Payload: line}
	}
	return InternalEvent{Type: "RAW", Payload: line}
}

func (c *VanishedClient) dispatch(event InternalEvent) {
	switch event.Type {
	case EvChatMsg:
		c.processIncomingChat(event.Payload)
	case EvKeyDelivery:
		c.handleKeyDelivery(event.Payload)
	case EvSystemInfo:
		c.handleSystemInfo(event.Payload)
	case EvHostPromoted:
		c.isHost = true
		ui.WriteToTerminal("HOST", "red", "STATUS: PROMOTED TO HOST (AES GEN)")
	default:
		ui.WriteToTerminal("NODE", "gray", event.Payload)
	}
}

func (c *VanishedClient) processIncomingChat(payload string) {
	// Limpieza flexible de CHAT_MSG:: o CHAT_MSG:
	cleanData := payload
	prefixes := []string{EvChatMsg + "::", EvChatMsg + ":"}
	for _, p := range prefixes {
		cleanData = strings.TrimPrefix(cleanData, p)
	}

	if strings.Contains(cleanData, ":") {
		parts := strings.SplitN(cleanData, ":", 2)
		userPart := strings.Trim(parts[0], "[] ")
		msgPart := strings.TrimSpace(parts[1])

		// INTENTO DE DESCIFRADO REAL
		if len(c.aesKey) > 0 {
			decrypted, err := cryptoutils.DecryptForChat(msgPart, c.aesKey)
			if err == nil {
				ui.WriteToTerminal(userPart, "darkred", decrypted)
				return
			}
		}
		// Si no hay llave o falla, mostramos lo que llegó (Junk Data)
		ui.WriteToTerminal(userPart, "gray", msgPart)
	} else {
		ui.WriteToTerminal("MSG", "gray", cleanData)
	}
}

func (c *VanishedClient) handleSystemInfo(payload string) {
	// Extraer contenido después de cualquier variante de SYSTEM_INFO
	content := payload
	if strings.Contains(payload, ":") {
		parts := strings.SplitN(payload, ":", 2)
		content = parts[1]
	}

	// 1. Detectar LISTA DE SALAS para la Sidebar
	if strings.Contains(payload, "LISTA DE SALAS") {
		lines := strings.Split(payload, "\n")
		var rooms []string
		for _, l := range lines {
			if strings.Contains(l, "•") {
				name := strings.TrimSpace(strings.ReplaceAll(l, "•", ""))
				rooms = append(rooms, name)
			}
		}
		if len(rooms) > 0 {
			ui.UpdateRoomsList(rooms)
		}
	}

	// 2. Detectar creación de sala para generar AES (DEBUGGER)
	if strings.Contains(payload, StatusRoomCreated) {
		newKey, err := cryptoutils.GenerateAESKey()
		if err == nil {
			c.aesKey = newKey
			c.isHost = true
			hash := sha256.Sum256(c.aesKey)
			ui.WriteToTerminal("AES_DEBUG", "green", fmt.Sprintf("AES GENERATED: %x", hash[:8]))
		}
	}

	ui.WriteToTerminal("SYSTEM", "yellow", content)
}

func (c *VanishedClient) handleKeyDelivery(line string) {
	// Formato esperado: KEY_DELIVERY:FROM:User:Key
	parts := strings.Split(line, ":")
	if len(parts) < 4 {
		return
	}

	subCmd := parts[1]
	sender := parts[2]
	data := parts[3]

	if subCmd == "FROM" {
		decKey, err := cryptoutils.DecryoptWithPrivateKey(data, c.privateKey)
		if err == nil {
			c.aesKey = decKey
			hash := sha256.Sum256(c.aesKey)
			ui.WriteToTerminal("SECURE", "green", fmt.Sprintf("KEY RECEIVED FROM %s [%x]", sender, hash[:8]))
		}
	} else if subCmd == "REQ_FROM" {
		ui.WriteToTerminal("KEY_REQ", "yellow", "SENDING KEY TO: "+sender)
		c.processKeyRequest(sender, data)
	}
}

func (c *VanishedClient) processKeyRequest(targetUser string, targetPubKey string) {
	if len(c.aesKey) == 0 {
		return
	}
	enc, err := cryptoutils.EncryptWithPublicKey(c.aesKey, targetPubKey)
	if err == nil {
		encB64 := base64.StdEncoding.EncodeToString(enc)
		fmt.Fprintf(c.conn, "/sendKey %s %s\n", targetUser, encB64)
	}
}
