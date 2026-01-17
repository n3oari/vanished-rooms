package storage

func (r *SQLiteRepository) CreateUser(u Users) error {
	query := `INSERT INTO users (uuid, name, password_hash) VALUES (?,?,?)`
	_, err := r.db.Exec(query, u.UUID, u.Username, u.PasswordHash)
	return err
}

func (r *SQLiteRepository) DeleteUser(u Users) error {
	query := `DELETE FROM users WHERE uuid = ?`
	_, err := r.db.Exec(query, u.UUID)
	return err
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
