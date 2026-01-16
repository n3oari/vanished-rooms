package storage

import (
	"database/sql"
	"log"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLHandler(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) CreateUser(u Users) error {
	query := `INSERT INTO users (uuid, name, password_hash, public_rsa_key) VALUES (?,?,?,?)`
	_, err := r.db.Exec(query, u.UUID, u.Username, u.PasswordHash, u.PublicRSAKey)
	return err
}

func (r *SQLiteRepository) DeleteUser(u Users) error {
	query := `DELETE FROM users WHERE uuid = ?`
	_, err := r.db.Exec(query, u.UUID)
	return err
}

// ------- //

func (r *SQLiteRepository) CreateAndJoinRoom(room Rooms, userUUID string) error {

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO rooms (uuid, name, password_hash) VALUES (?,?,?)`,
		room.UUID, room.Name, room.PasswordHash)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`INSERT INTO participants (uuid_room, uuid_user) VALUES (?,?)`,
		room.UUID, userUUID)
	if err != nil {
		tx.Rollback()
		return err
	}
	log.Println("[+] Transaction done! Room created and user joined successfully")
	return tx.Commit()
}

func (r *SQLiteRepository) DeleteRoom(u Rooms) error {
	query := `DELETE FROM rooms WHERE uuid = ?`
	_, err := r.db.Exec(query, u.UUID)
	if err != nil {
		return err
	}
	return nil
}
