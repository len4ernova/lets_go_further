package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// Initialized a new httprouter
	router := httprouter.New()

	// установить notFoundResponse в качестве пользовательского обработчика NotFound
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	// аналогично для не поддерживаемых методов
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// Register the relevant methods.
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovieHandler)
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.showMovieHandler)

	// REturn the httprouter instance.
	return app.recoverPanic(router)

}
