package data

import (
	"database/sql"
	"errors"
)

var (
	// used for multiple tables
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// model wrapper for easy autocomplete access
type Models struct {
	Movies MovieModel
	Users  UserModel
}

// constructor
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
		Users:  UserModel{DB: db},
	}
}
