package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/len4ernova/lets_go_further/internal/data"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new movie")
}

func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Retries a slice containing parametere names and values
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		//		http.NotFound(w, r)
		return
	}
	movie := data.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "Casabl",
		Runtime:   102,
		Genres:    []string{"drama", "romance", "war"},
		Version:   1,
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		// app.logger.Error(err.Error())
		// http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}
