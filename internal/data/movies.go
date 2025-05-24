package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// method for insert record into movie table
func (m MovieModel) Insert(movie *Movie) error {
	// insert statement (with weird string syntax)
	query := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version
	`
	// slice of placeholder params
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	// create a context with a 3 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// execute query and return result
	// we're writing the returned values back to the struct
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.ID, &movie.CreatedAt, &movie.Version)
}

// method for get record by id
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

	// create a context with a 3 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// execute query and pass values retrieved into the struct
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
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

// method to get all movies
func (m MovieModel) GetAll(title string, genres []string, filters Filters) ([]*Movie, error) {
	// query to get all movie records
	// the "where" clause allows title searching
	// the "and" where clause allows searching by genre(s)
	// the "order by" clause allows the user to order by column,
	// additionally orders by ID to ensure consistent ordering.
	// the last condition prevents cases where movies have a same column value,
	// e.g. both movies have the same year of 1999
	query := fmt.Sprintf(`
        SELECT id, created_at, title, year, runtime, genres, version
        FROM movies
	WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
	AND (genres @> $2 or $2 = '{}')
        ORDER BY %s %s, id ASC`, filters.sortColumn(), filters.sortDirection())

	// context with 3s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// execute query, returns sql.Rows resultset,
	// pass title and genres as placeholder param values
	rows, err := m.DB.QueryContext(ctx, query, title, pq.Array(genres))
	if err != nil {
		return nil, err
	}

	// close resultset; "dealloc"
	defer rows.Close()

	// array of movie pointers
	movies := []*Movie{}

	// iterate over the resultset to extract the data
	for rows.Next() {
		// tmp movie
		var movie Movie

		// scan values into the struct
		err := rows.Scan(
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			pq.Array(&movie.Genres),
			&movie.Version,
		)
		if err != nil {
			return nil, err
		}

		// add movie to slice
		movies = append(movies, &movie)
	}

	// check for any errors that occurred during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// success
	return movies, nil
}

// method for updating a record
func (m MovieModel) Update(movie *Movie) error {
	// set update query
	query := `
        UPDATE movies 
        SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
        WHERE id = $5 AND version = $6
        RETURNING version`

	// args slice to hold values
	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	// create a context with a 3 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	// return edit conflict error if necessary
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	// or not
	return nil
}

// method for deleting a record by id
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

	// create a context with a 3 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// execute query
	result, err := m.DB.ExecContext(ctx, query, id)
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
