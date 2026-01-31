package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/len4ernova/lets_go_further/internal/data"
	"github.com/len4ernova/lets_go_further/internal/validator"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	// I. использование io.ReadAll() для чтения тела запроса []byte
	// производительность ниже, чем у Decoder.
	// body, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	app.serverErrorResponse(w, r, err)
	// 	return
	// }
	// err = json.Unmarshal(body, &input)
	// if err != nil{
	// 	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
	// 	return
	// }

	// II. использование NewDecoder
	//err := json.NewDecoder(r.Body).Decode(&input)
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		//app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	// выполним проверки полей с помощью
	// предварительно скопируем данные в movie
	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}
	// Создаем новый валидатор
	v := validator.New()

	// вызываем ф-ию валидации
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// добавляем данные в БД
	err = app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// добавим заголовок Location чтобы клиент знал по какому адресу найти ресурс.
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	// Запишем ответ со статусом 201.
	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": movie}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
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
