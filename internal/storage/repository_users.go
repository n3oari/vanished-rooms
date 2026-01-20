package storage

import (
	"database/sql"
	"log"
)

func (r *SQLiteRepository) CreateUser(u Users) error {

	query := `INSERT INTO users (uuid, name, password_hash, public_rsa_key, salt) VALUES (?,?,?,?,?)`
	_, err := r.db.Exec(query, u.UUID, u.Username, u.PasswordHash, u.PublicRSAKey, u.Salt)
	if err != nil {
		return err
	}

	log.Printf("[!] Error en CreateUser: %v", err)

	return nil
}

func (r *SQLiteRepository) DeleteUser(u Users) error {
	query := `DELETE FROM users WHERE uuid = ?`
	_, err := r.db.Exec(query, u.UUID)
	return err
}

func (r *SQLiteRepository) GetPubKeyByUsername(userTarget string) (string, error) {
	pubKey := ""
	query := `SELECT public_rsa_key FROM users where name = ?`
	err := r.db.QueryRow(query, userTarget).Scan(&pubKey)
	if err != nil {
		return "", err
	}

	return pubKey, nil
}

func (r *SQLiteRepository) ListAllUsersInRoom(roomUUID string) ([]Users, error) {
	var users []Users

	query := `
	SELECT u.name
	FROM users u
	INNER JOIN participants p
	ON u.uuid = p.uuid_user
	WHERE p.uuid_room = ?`

	rows, err := r.db.Query(query, roomUUID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var user Users
		err := rows.Scan(&user.Username)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil

}

func (r *SQLiteRepository) RemoveParticipant(userUUID string, roomUUID string) error {
	query := `DELETE FROM participants WHERE uuid_user = ? AND uuid_room = ?`
	_, err := r.db.Exec(query, userUUID, roomUUID)
	return err
}

func (r *SQLiteRepository) PromoteNextHost(roomUUID, leavingUserUUID string) (string, error) {
	var nextUserUUID string
	query := `
    SELECT uuid FROM users
    WHERE uuid_current_room = ?
    AND uuid != ?
    ORDER BY joined_at ASC
    LIMIT 1`

	err := r.db.QueryRow(query, roomUUID, leavingUserUUID).Scan(&nextUserUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(`UPDATE users SET is_owner = 0 WHERE uuid = ?`, leavingUserUUID)
	if err != nil {
		tx.Rollback()
		return "", nil
	}

	_, err = tx.Exec(`UPDATE users SET is_owner = 1 WHERE uuid = ?`, nextUserUUID)
	if err != nil {
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}
	return nextUserUUID, nil
}

func (r *SQLiteRepository) SetUserAsOwner(userUUID string, status int) error {
	query := `UPDATE users SET is_owner = ? WHERE uuid = ?`
	_, err := r.db.Exec(query, status, userUUID)
	return err
}

// func (r  *SQLiteRepository) KickAllUsersFromRoom(roomUUID string ) error {}
