package main

import (
	"fmt"
	"net/http"
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
