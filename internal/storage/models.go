package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type Users struct {
	UUID            string    `db:"uuid"`
	Username        string    `db:"name"`
	PasswordHash    []byte    `db:"password_hash"`
	Salt            []byte    `db:"salt"`
	PublicRSAKey    string    `db:"public_rsa_key"`
	IsOwner         bool      `db:"is_owner"`
	CurrentRoomUUID string    `db:"uuid_current_room"`
	JoinedAt        time.Time `db:"joined_at"`
}

type Rooms struct {
	UUID         string `db:"uuid"`
	Name         string `db:"name"`
	PasswordHash []byte `db:"password_hash"`
	Salt         []byte `db:"salt"`
	Private      bool   `db:"private"`
}

type Participants struct {
	UUID     string `db:"uuid"`
	RoomUUID string `db:"uuid_room"`
	UserUUID string `db:"uuid_user"`
}

type Messages struct {
	UUID       string `db:"uuid"`
	Content    string `db:"content"`
	Timestamp  int64  `db:"timestamp"`
	RoomUUID   string `db:"room_uuid"`
	SenderUUID string `db:"sender_uuid"`
}

func InitDB(db *sql.DB) error {
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		uuid TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		password_hash BLOB NOT NULL,
		salt BLOB NOT NULL,
		public_rsa_key TEXT NOT NULL,
		is_owner BOOLEAN DEFAULT 0,
		uuid_current_room TEXT,
		joined_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(createUsersTable); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	createRoomsTable := `
	CREATE TABLE IF NOT EXISTS rooms (
		uuid TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		password_hash BLOB,
		salt BLOB,
		private INTEGER NOT NULL DEFAULT 0,
		max_users INTEGER NOT NULL DEFAULT 6
	);`

	if _, err := db.Exec(createRoomsTable); err != nil {
		return fmt.Errorf("failed to create rooms table: %w", err)
	}

	createParticipantsTable := `
	CREATE TABLE IF NOT EXISTS participants (
		uuid_room TEXT,
		uuid_user TEXT UNIQUE,
		PRIMARY KEY(uuid_room, uuid_user),
		CONSTRAINT fk_room FOREIGN KEY(uuid_room) REFERENCES rooms(uuid) ON DELETE CASCADE,
		CONSTRAINT fk_user FOREIGN KEY(uuid_user) REFERENCES users(uuid) ON DELETE CASCADE
	);`

	if _, err := db.Exec(createParticipantsTable); err != nil {
		return fmt.Errorf("failed to create participants table: %w", err)
	}

	return nil
}
