package storage

import "fmt"

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

func (r *SQLiteRepository) RemoveParticipant(userUUID string, roomUUID string) error {
	query := `DELETE FROM participants WHERE uuid_user = ? AND uuid_room = ?`
	_, err := r.db.Exec(query, userUUID, roomUUID)
	return err
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
