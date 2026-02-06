package main

import (
	"errors"
	"net/http"

	"github.com/len4ernova/lets_go_further/internal/data"
	"github.com/len4ernova/lets_go_further/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	// структура для хранения данных из тела запроса
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// парсим запрос
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	// копируем данные из запроса в структуру User
	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	// генерируем хеш пароля и сохраняем
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	// валидация структуры
	data.ValidateUser(v, user)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// добавить данные в БД
	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// вернуть клиенту ответ со  статусом 201
	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	//////////// mailer
	//методу Send() передать адрес пользовтеля, шаблон и пользоваетльские данные.
	/* TODO нет SMTP
	err = app.mailer.Send(user.Email, "user_welcome.tmpl", user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	*/
}
