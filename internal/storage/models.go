package storage

type Users struct {
	UUID           string `db:"uuid"`
	Username       string `db:"name"`
	Password_hash  string `db:"password_hash"`
	Public_rsa_key string `db:"public_rsa_key"`
}

type Rooms struct {
	UUID          string `db:"uuid"`
	name          string `db:"name"`
	password_hash string `db:"password_hash"`
}

type Participants struct {
	UUID      string `db:"uuid"`
	uuid_room string `db:"uuid_room"`
	uuid_user string `db:"uuid_user"`
}

type Messages struct {
	UUID        string `db:"uuid"`
	content     string `db:"content"`
	timestamp   int64  `db:"timestamp"`
	room_uuid   string `db:"room_uuid"`
	sender_uuid string `db:"sender_uuid"`
}
