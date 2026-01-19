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
