package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
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

		if roomName == "" || roomPass == "" {
			fmt.Fprintln(conn, "[!] Usage: /create -n <room_name> -p <room_password>\n")
			return
		}

		newRoom := storage.Rooms{
			UUID:         generateUUID(),
			Name:         roomName,
			PasswordHash: roomPass,
		}

		err := sv.SQLiteRepository.CreateAndJoinRoom(newRoom, User.UUID)
		if err != nil {
			fmt.Fprintf(conn, "[!] Database Error: %v\n", err)
			return
		} else {
			User.CurrentRoomUUID = newRoom.UUID
			fmt.Fprintf(conn, "[+] Room '%s' created successfully.\n", roomName)
		}

		sv.mu.Lock()
		sv.usersInRoom[User.UUID].CurrentRoomUUID = newRoom.UUID
		sv.mu.Unlock()

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
		if u, ok := sv.usersInRoom[User.UUID]; ok {
			u.CurrentRoomUUID = roomID
		}
		sv.mu.Unlock()

		w := bufio.NewWriter(conn)

		keyNotify := fmt.Sprintf("USER_JOINED:%s:%s\n", User.Username, User.PublicRSAKey)
		sv.broadcast(keyNotify, conn, roomID, sv.usersInRoom)

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
		for _, user := range users {
			sb.WriteString("- ")
			sb.WriteString(user.Username)
			sb.WriteString("\n")
		}

		_, err = conn.Write([]byte(sb.String()))
		if err != nil {
			log.Print("Error sending data to client")
		}

	case "/rooms":
		rooms, err := sv.SQLiteRepository.ListAllRooms()
		if err != nil {
			conn.Write([]byte("Error retrieving rooms\n"))
			return
		}

		var sb strings.Builder
		for _, room := range rooms {
			sb.WriteString("- ")
			sb.WriteString(room.Name)
			sb.WriteString("\n")
		}

		_, err = conn.Write([]byte(sb.String()))
		if err != nil {
			log.Print("Error sending data to client")
		}

	case "/leave-room":
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
		fmt.Printf("\n[DEBUG SERVIDOR] Recibido /sendKey para: %s\n", targetName)
		fmt.Printf("[DEBUG SERVIDOR] Payload RSA (B64): %s\n", encryptedKeyB64)

		sv.mu.Lock()
		var targetConn net.Conn

		for _, u := range sv.usersInRoom {
			if u.Username == targetName {
				targetConn = sv.clients[u.UUID]
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
		fmt.Fprintln(conn, "\nopenssl genrsa -out privada.pem 2048 -> Generate private key")

	case "/quit":
		fmt.Fprintln(conn, "[!] ¡¡ BYEE !!")
		conn.Close()
		return

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
