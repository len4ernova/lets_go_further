package main

import (
	"fmt"
	"net/http"
)

// logError - запись сообщения об ошибке.
func (app *application) logError(r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	app.logger.Error(err.Error(), "method", method, "uri", uri)

}

// errorResponse - универсальный метод отправки сообщения об ошибке.
func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message any) {
	env := envelope{"error": message}

	err := app.writeJSON(w, status, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

// serverErrorResponse - метод используется когда сервер столкнулся с непредвиденной ошибкой
// Метод используется для отправки клиенту 500.
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)

	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

// notFoundResponse - метод используется для отправки клиенту 404.
func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

// methodNotAllowedResponse - метод используется для отправки клиенту 405.
func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// badRequestResponse - метод используется для отправки клиенту 400.
func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

// failedValidationResponse - метод используется для отправки клиенту 422.
// Возникает в слечае, если данные отправленные клиентом были распознаны,
// но при обработке произошла ошибка (в т.ч. в случае не валидности)
// Тип errors совпадает с map Validator
func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

// editConflictResponse - метод используется для отправки клиенту 409.
// Возникает в случае конфликта при записи данных измененных.
func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorResponse(w, r, http.StatusConflict, message)
}

// rateLimitExceededResponse - метод используется для отправки клиенту 429 «Слишком много запросов»
func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceed"
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

// invalidCreditialsResponse - метод используется для отправки клиенту 401 «Не авторизован»
func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication creditials"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// invalidAuthenticationTokenResponse - метод используется для отправки клиенту 401 «Не авторизован»,
// если метод Authentication представлен, но имеет некорректеный формат или недопустимые значения.
func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	// напоминаем клиенту что хотим получить "Bearer ..."
	w.Header().Set("WWW-Authenticate", "Bearer")

	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// authenticationRequireResponse - метод отправляет клиенту 401, если клиент запрашивает доступ к ресурсу без аутентификациии.
func (app *application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

// inactiveAccountREsponse - метод отправляет клиенту 403, если клиенту не разрешен доступ к ресурсу.
func (app *application) inactiveAccountREsponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}
