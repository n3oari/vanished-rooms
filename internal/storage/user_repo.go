package storage

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
