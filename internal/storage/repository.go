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
