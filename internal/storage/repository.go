package storage

import (
	"database/sql"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLHandler(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) CreateUser(u Users) error {
	query := `INSERT INTO users (uuid, name , public_rsa_key,password_hash) VALUES (?,?,?,?)`
	_, err := r.db.Exec(query, u.UUID, u.Username, u.Public_rsa_key)
	return err
}

func (r *SQLiteRepository) DeleteUser(u Users) error {
	query := `DELETE FROM users (uuid) VALUES (?)`
	_, err := r.db.Exec(query, u.UUID)
	return err
}

// ------- //

func (r *SQLiteRepository) CreateRooms(u Users) error {
	query := `INSERT INTO rooms (uuid, name , password_hash) VALUES (?,?,?)`
	_, err := r.db.Exec(query, u.UUID, u.Username, u.Password_hash)
	return err
}

func (r *SQLiteRepository) DeleteRoom(u Users) error {
	query := `DELETE FROM userss (uuid) VALUES (?)`
	_, err := r.db.Exec(query, u.UUID)
	return err
}
