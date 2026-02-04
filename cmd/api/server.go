package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// когда мы получаем сигнал SIGINT или SIGTERM, мы даем нашему серверу команду прекратить прием новых HTTP-запросов и даем всем выполняющимся запросам «льготный период» в 30 секунд на завершение работы до того, как приложение будет остановлено.
func (app *application) server() error {
	//  HTTP-сервер с  настройками
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}
	//  канал shutdownError будем использовать для получения любых ошибок, возвращаемых функцией корректного завершения работы Shutdown()
	shutdownError := make(chan error)

	// старт фоновой горутины
	go func() {
		//канал выхода, который будет передавать значения os.Signal
		quite := make(chan os.Signal, 1)
		//  signal.Notify() отслеживает входящие сигналы SIGINT и SIGTERM и перенаправляет их в канал выхода.
		//  Любые другие сигналы не будут перехватываться signal.Notify() и будут обрабатываться по умолчанию
		signal.Notify(quite, syscall.SIGINT, syscall.SIGTERM)

		//Считываем сигнал с канала quit
		s := <-quite

		// сообщение о том, что сигнал был обработан
		app.logger.Info("caught signal", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// Вызываем Shutdown() на нашем сервере, передавая только что созданный контекст.
		// Shutdown() вернет nil, если корректное завершение работы прошло успешно, или
		//  ошибку (которая может возникнуть из-за проблем с закрытием прослушивателей или
		//  из-за того, что завершение работы не было завершено до истечения 30-секундного
		//  срока действия контекста). Мы передаем это возвращаемое значение в канал shutdownError.
		shutdownError <- srv.Shutdown(ctx)
	}()

	// Start the HTTP server.
	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config)
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	// В противном случае мы ожидаем получения возвращаемого значения от Shutdown() по каналу
	//  shutdownError. Если возвращаемое значение — ошибка, значит, при корректном
	//  завершении работы возникла проблема, и мы возвращаем ошибку.
	err = <-shutdownError
	if err != nil {
		return err
	}
	// На этом этапе мы знаем, что корректное завершение работы прошло успешно, и записываем в журнал сообщение «Сервер остановлен».
	app.logger.Info("stopped server", "addr", srv.Addr)
	return nil
}
