package storage

import (
	"database/sql"
	"fmt"
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

func (r *SQLiteRepository) JoinRoom(userUUID, nameRoom, passRoom string) (string, error) {

	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	var roomUUID string
	querySelect := (`SELECT uuid  FROM rooms WHERE name = ? AND password_hash = ?`)
	err = tx.QueryRow(querySelect, nameRoom, passRoom).Scan(&roomUUID)

	if err != nil {
		tx.Rollback()
		return "", err
	}

	queryInsert := `INSERT INTO participants (uuid_room, uuid_user) VALUES (?,?)`
	_, err = tx.Exec(queryInsert, roomUUID, userUUID)

	if err != nil {
		tx.Rollback()
		return "", err
	}
	log.Printf("[+] Transaction done! Room created and user joined successfully\n roomUUID -> %s", roomUUID)
	err = tx.Commit()
	return roomUUID, err

}

func (r *SQLiteRepository) DeleteRoom(roomUUID string) error {
	query := `DELETE FROM rooms WHERE uuid = ?`
	_, err := r.db.Exec(query, roomUUID)
	if err != nil {
		return err
	}
	return nil
}

func (r *SQLiteRepository) DeleteRoomIfEmpty(roomUUID string) (bool, error) {
	var count int
	queryCount := `SELECT COUNT(*) FROM participants WHERE uuid_room = ?`
	err := r.db.QueryRow(queryCount, roomUUID).Scan(&count)
	if err != nil {
		return false, err
	}

	if count == 0 {
		err := r.DeleteRoom(roomUUID)
		if err != nil {
			return false, err
		}
		fmt.Printf("[+] Room with UUID %s deleted as it was empty.\n", roomUUID)
		return true, nil
	}
	return false, nil
}

func (r *SQLiteRepository) RemoveParticipant(userUUID string, roomUUID string) error {
	query := `DELETE FROM participants WHERE uuid_user = ? AND uuid_room = ?`
	_, err := r.db.Exec(query, userUUID, roomUUID)
	return err
}
