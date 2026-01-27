package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type envelope map[string]any

// readIDParam - получить "id" из контектста запроса и преобразовать в int.
func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}
	return id, nil
}

// writeJSON - помогает вв отправке ответов.
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	//
	// js, err := json.Marshal(data)
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	js = append(js, '\n')

	// добавить заголовки
	for key, value := range headers {
		w.Header()[key] = value
	}
	// или
	// maps.Insert(w.Header(), maps.All(headers))

	// загоовок "Content-Type: application/json", код ответа и json данные
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}
