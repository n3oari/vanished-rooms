package storage

type Users struct {
	UUID         string `db:"uuid"`
	Username     string `db:"name"`
	PasswordHash string `db:"password_hash"`
	PublicRSAKey string `db:"public_rsa_key"`
}

type Rooms struct {
	UUID         string `db:"uuid"`
	Name         string `db:"name"`
	PasswordHash string `db:"password_hash"`
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
