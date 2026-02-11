package main

import (
	"context"
	"net/http"

	"github.com/len4ernova/lets_go_further/internal/data"
)

// создадим пользовательский тип для контекста запроса
type contextKey string

// используем  ключ user для получениятнфо о пользователе в контексте запроса
const userContextKey = contextKey("user")

// contextSetUser - метод вернет копию запроса с данными User добавленными в контекст.
// userContextKey используется как ключ.
func (app *application) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// ContextGetUser - извлекаем struct User  из контекста.
// Исп-ся во вспомогательных методах, когда ожидаем наличия структуры.
// Если структуры нет, то пиника.
func (app *application) ContextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("mission user value in request context")
	}

	return user
}
