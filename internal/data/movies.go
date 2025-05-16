package data

import (
	"database/sql"
	"errors"
	"time"

	"github.com/Bekian/greenlight/internal/validator"

	"github.com/lib/pq"
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
	// insert statement (with weird string syntax)
	query := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version
	`
	// slice of placeholder params
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	// execute query and return result
	// we're writing the returned values back to the struct
	return m.DB.QueryRow(query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

// placeholder method for get record by id
func (m MovieModel) Get(id int64) (*Movie, error) {
	// validate positive integer ID
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	// query for retrieving the movie data
	query := `
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE id = $1
	`

	// struct to hold retrieved data
	var movie Movie

	// execute query and pass values retrieved into the struct
	err := m.DB.QueryRow(query, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)

	// handle errors
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &movie, nil
}

// placeholder method for updating a record
func (m MovieModel) Update(movie *Movie) error {
	// set update query
	query := `
		UPDATE movies
		SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
		WHERE id = $5
		RETURNING version
	`

	// args slice to hold values
	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
	}

	return m.DB.QueryRow(query, args...).Scan(&movie.Version)
}

// placeholder method for deleting a record by id
func (m MovieModel) Delete(id int64) error {
	// validate positive integer ID
	if id < 1 {
		return ErrRecordNotFound
	}

	// set delete query
	query := `
		DELETE FROM movies
		WHERE id = $1
	`

	// execute query
	result, err := m.DB.Exec(query, id)
	if err != nil {
		return err
	}

	// get number of rows effected to validate query
	rowsEffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// run validate check
	if rowsEffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}
