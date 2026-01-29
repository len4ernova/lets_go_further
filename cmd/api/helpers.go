package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

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

// readJSON - декодирование JSON из запроса и анализ ошибок.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// http.MaxBytesReader() ограничение размера тела запроса 1,048,576 bytes (1MB)
	r.Body = http.MaxBytesReader(w, r.Body, 1_048_576)
	// декодируем и вызовем DisallowUnknowFields (вызовет ошибку, если в запросе пользователя к-л поле не соотв-ет целевому обекту)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	// декодирование тела запроса
	err := dec.Decode(dst)
	if err != nil {
		// анализ причин ошибок
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		// перехват синтаксических ошибок
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		// иногда при синтаксической ошибке возвращается io.ErrUnexpectedEOF.
		//  https://github.com/golang/go/issues/25956
		// поэтому проверим ее с пом.errors.Is
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		// перехват ошибки - значение JSON имеет несоответствие типу целевого объекта
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		// перехват ошибки - пустое тело запроса
		case errors.Is(err, io.EOF):
			return errors.New("body must be empty")

		// перехват ошибки - если JSON содержит поле, которое не может быть сопоставлено целевому
		// Note that there's an open issue at https://github.com/golang/go/issues/29035 regarding turning this
		// into a distinct error type in the future
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fileName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fileName)

		// перехват ошибки - тело запроса превысило лимит.
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be large than %d bytes", maxBytesError.Limit)

		// перехват ошибки - передача в Decode чего-либо кроме ненелевого указателя
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}
	// Вызовем Decode еще раз - для отслеживания дополнительного json.
	// Если в запросе больше не было данных, то вернет EOF.
	// Иначе сформируем сообщение об ошибке.
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
}
