package storage

import "time"

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
	MaxUsers     int    `db:"max_users"`
}

type Participants struct {
	UUID     string `db:"uuid"`
	RoomUUID string `db:"uuid_room"`
	UserUUID string `db:"uuid_user"`
}
