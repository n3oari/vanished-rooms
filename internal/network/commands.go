package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"vanished-rooms/internal/cryptoutils"
	"vanished-rooms/internal/storage"
)

func (sv *Server) HandleInternalCommand(conn net.Conn, User *storage.Users, msg string) {
	parts := strings.Fields(msg)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "/create":
		roomName := extractFlag(msg, "-n")
		roomPass := extractFlag(msg, "-p")
		isPrivate := extractFlag(msg, "--private")

		roomName = strings.TrimSpace(roomName)
		roomPass = strings.TrimSpace(roomPass)
		isPrivate = strings.TrimSpace(isPrivate)
		privacyChoice := false

		if roomName == "" || roomPass == "" || isPrivate == "" {
			fmt.Fprintln(conn, "[!] Usage: /create -n <room_name> -p <room_password> --private <y/n>")
			break
		}

		salt, err := cryptoutils.GenerarSalt()
		if err != nil {
			fmt.Fprintf(conn, "[!] Error generating salt: %v\n", err)
			break
		}

		hash := cryptoutils.HashPassword(roomPass, salt)

		if isPrivate == "y" {
			privacyChoice = true
		} else if isPrivate == "n" {
			privacyChoice = false
		} else {
			fmt.Fprintln(conn, "[!] Invalid privacy option. Use 'y' or 'n'.")
			break
		}

		newRoom := storage.Rooms{
			UUID:         generateUUID(),
			Name:         roomName,
			PasswordHash: hash,
			Salt:         salt,
			Private:      privacyChoice,
		}

		err = sv.SQLiteRepository.CreateAndJoinRoom(newRoom, User.UUID)
		if err != nil {
			fmt.Fprintf(conn, "[!] Database Error: %v\n", err)
			break
		}

		User.CurrentRoomUUID = newRoom.UUID
		User.IsOwner = true

		sv.mu.Lock()
		if u, exists := sv.UsersInRoom[User.UUID]; exists {
			u.CurrentRoomUUID = newRoom.UUID
			u.IsOwner = true
		}
		sv.mu.Unlock()

		fmt.Fprintf(conn, "[+] Room '%s' created successfully. You are the host.\n", roomName)

	case "/join":
		roomName := extractFlag(msg, "-n")
		roomPass := extractFlag(msg, "-p")
		if roomName == "" || roomPass == "" {
			fmt.Fprint(conn, "[!] Usage: /join -n <room_name> -p <room_password>\n")
			return
		}

		roomID, err := sv.SQLiteRepository.JoinRoom(User.UUID, roomName, roomPass)
		if err != nil {
			fmt.Fprintf(conn, "[!] Failed to join room: %v\n", err)
			return
		}

		User.CurrentRoomUUID = roomID
		sv.mu.Lock()
		sv.UsersInRoom[User.UUID] = User
		sv.mu.Unlock()

		fmt.Printf("\n[DEBUG SERVIDOR] Usuario %s se une a %s\n", User.Username, roomName)

		w := bufio.NewWriter(conn)

		keyNotify := fmt.Sprintf("USER_JOINED:%s:%s\n", User.Username, User.PublicRSAKey)
		sv.broadcast(keyNotify, conn, roomID, sv.UsersInRoom)

		fmt.Fprintf(w, "[+] Success! You have joined the room: %s\n", roomName)
		w.Flush()

	case "/users":
		if User.CurrentRoomUUID == "" {
			conn.Write([]byte("[!] You need to be in a room. Use <join>\n"))
			return
		}
		users, err := sv.SQLiteRepository.ListAllUsersInRoom(User.CurrentRoomUUID)
		if err != nil {
			return
		}

		var sb strings.Builder
		sb.WriteString("\n=== USER LIST ===\n")
		for _, user := range users {
			fmt.Fprintf(&sb, " • %s\n", user.Username)
		}
		sb.WriteString("======================\n")

		_, err = conn.Write([]byte(sb.String()))
		if err != nil {
			log.Print("Error sending data to client")
		}

	case "/rooms":
		rooms, err := sv.SQLiteRepository.ListAllRooms()
		if err != nil {
			log.Printf("Error al listar salas: %v", err)
			_, _ = conn.Write([]byte("[!] Error al recuperar las salas\n"))
			return
		}

		if len(rooms) == 0 {
			_, _ = conn.Write([]byte("[i] No hay salas disponibles.\n"))
			return
		}

		var sb strings.Builder
		sb.WriteString("\n=== LISTA DE SALAS ===\n")
		for _, room := range rooms {
			fmt.Fprintf(&sb, " • %s\n", room.Name)
		}
		sb.WriteString("======================\n")

		_, err = conn.Write([]byte(sb.String()))
		if err != nil {
			log.Printf("Error enviando lista: %v", err)
		}
	case "/leave-room":

		sv.mu.Lock()
		u, exists := sv.UsersInRoom[User.UUID]
		if !exists {
			sv.mu.Unlock()
			break
		}
		roomID := u.CurrentRoomUUID
		wasOwner := u.IsOwner
		sv.mu.Unlock()

		if wasOwner {
			newHostUUID, err := sv.SQLiteRepository.PromoteNextHost(roomID, User.UUID)
			if err == nil && newHostUUID != "" {
				NotifyPromotion(sv, newHostUUID)
			}
		}

		err := sv.SQLiteRepository.LeaveRoomAndDeleteRoomIfEmpty(User.UUID, User.CurrentRoomUUID)
		if err != nil {
			fmt.Printf("[!] Error leaving the room ", err)
			conn.Write([]byte("[!] Error leaving the room\n"))
			return
		}

		User.CurrentRoomUUID = ""
		conn.Write([]byte("[+] You leaved the room successfully\n"))

	case "/sendKey":
		//this is a server internal command, not for users
		//TO-DO: deny to users
		parts := strings.SplitN(msg, " ", 3)
		if len(parts) < 3 {
			return
		}

		targetName := parts[1]
		encryptedKeyB64 := parts[2]

		sv.mu.Lock()
		var targetConn net.Conn

		for _, u := range sv.UsersInRoom {
			if u.Username == targetName {
				targetConn = sv.Clients[u.UUID]
				break
			}
		}

		sv.mu.Unlock()

		if targetConn != nil {
			fmt.Fprintf(targetConn, "KEY_DELIVERY:%s:%s\n", User.Username, encryptedKeyB64)
			fmt.Printf("[DEBUG SERVIDOR] KEY_DELIVERY enviado a %s\n", targetName)
		}

	case "/help":
		fmt.Fprintln(conn, "[?] Available commands:")
		fmt.Fprintln(conn, "    /create -n <name> -p <pass>  -> Create a new room and join")
		fmt.Fprintln(conn, "    /rooms                       -> List all rooms")
		fmt.Fprintln(conn, "    /join -n <name> -p <pass>    -> Join a room")
		fmt.Fprintln(conn, "    /leave-room                  -> Leave the room ")
		fmt.Fprintln(conn, "    /users                       -> List all users (you need to be in a room")
		fmt.Fprintln(conn, "    /help                        -> Show help menu")
		fmt.Fprintln(conn, "    /quit                        -> Disconnect and Remove user permanetly (you can also use control + C")
		fmt.Fprintln(conn, "\n\nopenssl genrsa -out privada.pem 2048 -> Generate private key")

	case "/quit":

	default:
		fmt.Fprintf(conn, "[!] Unknown command: %s. Type /help for info.\n", parts[0])
	}
}

// idx+1 < len(parts) is for security: ensures the flag isn't the last word
// and prevents an "index out of range" panic.
func extractFlag(msg, flag string) string {
	parts := strings.Fields(msg)
	for idx, part := range parts {
		if part == flag && idx+1 < len(parts) {
			return parts[idx+1]
		}
	}
	return ""
}
