package main

import (
	"errors"
	"net/http"
	"time"

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
	// err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	// if err != nil {
	// 	app.serverErrorResponse(w, r, err)
	// }

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	app.logger.Info("token user", token)

	//////////// mailer
	//методу Send() передать адрес пользовтеля, шаблон и пользоваетльские данные.
	// TODO нет SMTP
	//  запустить горутину анонимной ф-ии выполняющей отправку сообщения

	// go func() {
	// 	// ф-ия обработки паники. Т.к. паника в горутине приведет к выходу из приложения.
	// 	defer func() {
	// 		if err := recover(); err != nil {
	// 			app.logger.Error(fmt.Sprintf("%v", err))
	// 		}
	// 	}()

	// 	//отправка приветственного сообщения
	// 	err = app.mailer.Send(user.Email, "user_welcome.tmpl", user)
	// 	if err != nil {
	// 		// app.serverErrorResponse(w, r, err)
	// 		// return
	// 		// не исп-ем serverErrorResponse, т.к. это вызовет дополнительную отправку клиенту, а мы хотим отправить 202.
	// 		app.logger.Error(err.Error())
	// 	}
	// }()

	// замениv вызов фоновой горутины на вызов background()
	app.background(func() {
		//map содержит фрагменты данных токет и ИД
		data := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			// не исп-ем serverErrorResponse, т.к. это вызовет дополнительную отправку клиенту, а мы хотим отправить 202.
			app.logger.Error(err.Error())
		}
	})

	// клиенту отправим ответ 202 - обработка начата, не завершена.
	err = app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	//*/
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	// из тела запроса заберем активационный токен
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// получить данные пользователя по предоставленному токену. Если записи нет, то клиент получит ошибку.
	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// обновить - присвоить статус активирован
	user.Activated = true

	// сохранить в БД, проверив конфликт редактирования
	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// удалить токен
	err = app.models.Tokens.DeleteToken(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// если все прошло успешно, удалить все токены активации пользователя
	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
