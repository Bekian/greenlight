package data

import (
	"database/sql"
	"time"

	"github.com/Bekian/greenlight/internal/validator"
)

// the hyphen directive always omits
// the omitzero directive omits when zero value
type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"` // Use the - directive
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitzero"`    // Add the omitzero directive
	Runtime   Runtime   `json:"runtime,omitzero"` // Add the omitzero directive
	Genres    []string  `json:"genres,omitzero"`  // Add the omitzero directive
	Version   int32     `json:"version"`
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")
}

// connection pool wrapper
type MovieModel struct {
	DB *sql.DB
}

// placeholder method for insert record into movie table
func (m MovieModel) Insert(movie *Movie) error {
	return nil
}

// placeholder method for get record by id
func (m MovieModel) Get(id int64) (*Movie, error) {
	return nil, nil
}

// placeholder method for updating a record
func (m MovieModel) Update(movie *Movie) error {
	return nil
}

// placeholder method for deleting a record by id
func (m MovieModel) Delete(id int64) error {
	return nil
}
