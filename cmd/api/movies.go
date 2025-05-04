package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Bekian/greenlight/internal/data"
)

// temp handler to create a new movie
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	// target struct to decode request info into
	var input struct {
		Title   string   `json:"title"`
		Year    int32    `json:"year"`
		Runtime int32    `json:"runtime"`
		Genres  []string `json:"genres"`
	}

	// decoder to pull response into struct
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// temp dump
	fmt.Fprintf(w, "%+v\n", input)
}

// temp handler to show a movie
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// get the id
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// create a movie instance with dummy data
	movie := data.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Runtime:   102,
		Genres:    []string{"drama", "romance", "war"},
		Version:   1,
	}

	// write the json with an envelope
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
