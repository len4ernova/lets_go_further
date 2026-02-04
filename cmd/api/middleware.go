package main

import (
	"fmt"
	"net/http"

	"golang.org/x/time/rate"
)

// recoverPanic - обработка паники.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// создадим ф-ию котоая всегда выполняется в случае паники
		defer func() {
			// используем встроенную ф-ию восстановления, которая проверяет произощла паника или нет
			if err := recover(); err != nil {
				// в случае паники: установить заголовок "Connection: close"/
				// Это послужит триггером для закрытия соединния после отправки
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// rateLimit - ограничитель запросов. корзина с токенами пополняется со скоростью два токена в секунду.
func (app *application) rateLimit(next http.Handler) http.Handler {
	// инициализируем новый ограничитель запросов в среднем 2 запроса/сек, макс 4
	limiter := rate.NewLimiter(2, 4)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Вызываем метод limiter.Allow(), чтобы проверить, разрешен ли запрос.
		// Если нет, то мы вызываем вспомогательный метод rateLimitExceededResponse()
		if !limiter.Allow() {
			app.rateLimitExceededResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
