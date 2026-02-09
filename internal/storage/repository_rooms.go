package storage

import (
	"bytes"
	"database/sql"
	"errors"
	"strings"
	"vanished-rooms/internal/cryptoutils"
)

func (r *SQLiteRepository) CreateAndJoinRoom(room Rooms, userUUID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO rooms (uuid, name, password_hash, salt, private) VALUES (?,?,?,?,?)`,
		room.UUID, room.Name, room.PasswordHash, room.Salt, room.Private)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO participants (uuid_room, uuid_user) VALUES (?,?)`,
		room.UUID, userUUID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE users SET 
		is_owner = 1, uuid_current_room = ?, joined_at = CURRENT_TIMESTAMP WHERE uuid = ?`,
		room.UUID, userUUID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SQLiteRepository) JoinRoom(userUUID, nameRoom, passRoom string) (string, string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()

	var roomUUID string
	var storedHash, salt []byte
	var isPrivate bool
	var maxUsers int

	var count int
	queryCount := `SELECT COUNT(*) FROM participants WHERE uuid_room = ?`
	err = tx.QueryRow(queryCount, roomUUID).Scan(&count)
	if err != nil {
		return "", "", err
	}

	if count >= maxUsers {
		return "", "", errors.New("room is full")
	}

	querySelect := `SELECT uuid, password_hash, salt, private,maxUsers  FROM rooms WHERE name = ?`
	err = tx.QueryRow(querySelect, nameRoom).Scan(&roomUUID, &storedHash, &salt, &isPrivate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", errors.New("room not found")
		}
		return "", "", err
	}

	if isPrivate {
		inputHash := cryptoutils.HashPassword(passRoom, salt)
		if !bytes.Equal(inputHash, []byte(storedHash)) {
			return "", "", errors.New("invalid credentials")
		}
	}

	var hostUUID string
	queryHost := `SELECT uuid FROM users WHERE uuid_current_room = ? AND is_owner = 1 LIMIT 1`
	err = tx.QueryRow(queryHost, roomUUID).Scan(&hostUUID)
	if err != nil {
		hostUUID = ""
	}

	queryInsert := `INSERT INTO participants (uuid_room, uuid_user) VALUES (?,?)`
	if _, err = tx.Exec(queryInsert, roomUUID, userUUID); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return "", "", errors.New("already in room")
		}
		return "", "", err
	}

	queryUpdate := `UPDATE users SET uuid_current_room = ?, is_owner = 0, joined_at = CURRENT_TIMESTAMP WHERE uuid = ?`
	if _, err = tx.Exec(queryUpdate, roomUUID, userUUID); err != nil {
		return "", "", err
	}

	if err = tx.Commit(); err != nil {
		return "", "", err
	}

	return roomUUID, hostUUID, nil
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
	var count int
	err = tx.QueryRow(`SELECT COUNT(*) FROM participants WHERE uuid_room = ?`, roomUUID).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = tx.Exec(`DELETE FROM rooms WHERE uuid = ?`, roomUUID)
		if err != nil {
			return err
		}
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
	query := `SELECT name FROM rooms WHERE private = 0`
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

func (r *SQLiteRepository) GetRoomCredentials(roomName string) (hash []byte, salt []byte, err error) {
	query := `SELECT password_hash, salt FROM rooms WHERE name = ?`

	err = r.db.QueryRow(query, roomName).Scan(&hash, &salt)
	if err != nil {
		return nil, nil, err
	}
	return hash, salt, nil
}

func (r *SQLiteRepository) ListPublicRooms() ([]string, error) {
	query := `SELECT name FROM rooms WHERE private = 0`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			rooms = append(rooms, name)
		}
	}
	return rooms, nil
}

func (r *SQLiteRepository) LimitUsersInRoom(roomUUID string) error {
	count := 0
	query := `SELECT COUNT(*) FROM participants WHERE uuid_room = ?`
	err := r.db.QueryRow(query, roomUUID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 8 {
		return errors.New("room user limit exceeded")
	}
	return nil
}
func (r *SQLiteRepository) GetRoomByName(name string) (string, error) {
	var uuid string
	query := `SELECT uuid FROM rooms WHERE name = ?`

	err := r.db.QueryRow(query, name).Scan(&uuid)
	if err != nil {
		return "", err
	}
	return uuid, nil
}

func (r *SQLiteRepository) PurgeEverything() error {
	_, err := r.db.Exec(`DELETE FROM participants`)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`DELETE FROM rooms`)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`DELETE FROM users`)
	return err
}
