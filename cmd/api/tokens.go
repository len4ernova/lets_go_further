package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/len4ernova/lets_go_further/internal/data"
	"github.com/len4ernova/lets_go_further/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// поиск пользователя по email.
	// Если не найден, то возврат кода 401 Не авторизован.
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// проверка на совпадение пароля
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// если пароли не совпадают
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// генертруем токен - 24 часа срок действия, область - 'authentication'
	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// закодируем токен и отправим 201
	err = app.writeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Сгенерируйте токен для сброса пароля и отправьте его по эл.почте
func (app *application) createPasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	// проанализировать и проверить адрес эл.почты
	var input struct {
		Email string `json:"email"`
	}
	err := app.readJSON(w, r, input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// получить соотв-ую запись пользователя в БД
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("email", "no matching email address found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// если пользователь не активирован вернуть ошибку
	if !user.Activated {
		v.AddError("email", "user account must be activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// создайте токен для сброса пароля на 45 мин
	token, err := app.models.Tokens.New(user.ID, 45*time.Minute, data.ScopePasswordReset)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// отправьте пользователю токен со сбросом пароля
	app.background(func() {
		data := map[string]any{
			"passwordResetToken": token.Plaintext,
		}

		// используем адрес хранящийся в БД
		err = app.mailer.Send(user.Email, "token_password_reset.tmpl", data)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})
	// отправить клиенту 202 и подтврждающее сообщение
	env := envelope{"message": "an email will be sent to you containing password reset instructions"}

	err = app.writeJSON(w, http.StatusAccepted, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// создание дополнительных токенов активации, если токен забыл активироваться и токен истек.
func (app *application) createActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	// прсинг и валидация email
	var input struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// пробуем получить запись пользователя
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("email", "no matching email address found")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Вернуть ошибку, если пользователь уже активирован
	if user.Activated {
		v.AddError("email", "user has already been activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// создать токен активации
	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	//отпраить пользователю email с доп. токеном активации
	app.background(func() {
		data := map[string]any{
			"activationToken": token.Plaintext,
		}
		// отправить пользователь по адресу хранящемуся в email
		err = app.mailer.Send(user.Email, "token_activation.tmpl", data)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

	// отправить пользователю 202 и сообщение подтверждающее отправку
	env := envelope{"message": "an email will be sent to you containing activation instructions"}
	err = app.writeJSON(w, http.StatusAccepted, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
