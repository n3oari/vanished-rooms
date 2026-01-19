package storage

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

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

	_, err = tx.Exec(`
	UPDATE users SET 
	is_owner = 1, uuid_current_room = ?, joined_at = CURRENT_TIMESTAMP  WHERE uuid = ?`,
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
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return "", errors.New("You already in a room, use /leave first")
		}
		return "", errors.New("error al unirse a la sala")
	}

	queryUpdate := `
	UPDATE users SET
	uuid_current_room = ?,
	is_owner = 0,
	joined_at = CURRENT_TIMESTAMP
	WHERE uuid = ?`

	_, err = tx.Exec(queryUpdate, roomUUID, userUUID)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	log.Printf("[+] Transaction done! Room created and user joined successfully\n roomUUID -> %s", roomUUID)
	err = tx.Commit()
	return roomUUID, err

}

func (r *SQLiteRepository) LeaveRoomAndDeleteRoomIfEmpty(userUUID, roomUUID string) error {

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM participants WHERE uuid_user = ? AND uuid_room = ?`, userUUID, roomUUID)
	if err != nil {
		return err
	}

	count := 0
	err = tx.QueryRow(`SELECT COUNT(*) FROM participants WHERE uuid_room = ?`, roomUUID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = tx.Exec(`DELETE FROM rooms WHERE uuid = ?`, roomUUID)
		if err != nil {
			return err
		}
		fmt.Printf("[-] Room %s deleted for being empty\n", roomUUID)
	}

	return tx.Commit()
}

func (r *SQLiteRepository) DeleteRoom(roomUUID string) error {
	query := `DELETE FROM rooms WHERE uuid = ?`
	_, err := r.db.Exec(query, roomUUID)
	if err != nil {
		return err
	}
	return nil
}

func (r *SQLiteRepository) ListAllRooms() ([]Rooms, error) {
	var rooms []Rooms
	query := `SELECT name FROM rooms`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var room Rooms
		err := rows.Scan(&room.Name)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return rooms, nil
}
