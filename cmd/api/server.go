package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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

		// Закрыть приложение с кодом состояния 0 (успех)
		os.Exit(0)
	}()

	// Start the HTTP server.
	app.logger.Info("starting server", "addr", srv.Addr, "env", app.config)
	return srv.ListenAndServe()
}
