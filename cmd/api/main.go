package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/len4ernova/lets_go_further/internal/data"
	_ "github.com/lib/pq"
)

const version = "1.0.0" // version app

// configuration settings for app
type config struct {
	port int
	env  string
	db   struct {
		dsn string
		// настройка пула соединений
		maxOpenConns int           // PostgreSQL макс открытых соединений
		maxIdleConns int           // PostgreSQL макс неактивных соединений
		maxIdleTime  time.Duration // PostgreSQL продолжительность неакт. соединения
	}
}

// an application struct to hold the dependencies for our HTTP handlers, helpers, and middleware
type application struct {
	config config
	logger *slog.Logger
	models data.Models
}

func main() {
	var cfg config

	// Read the value of the port and env command-line flags into the config struct.
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	//flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://user://pass@localhost/greenlight", "PostgeSQL DSN")
	// flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://user:pass@localhost/greenlight?sslmode=disable", "PostgeSQL DSN")
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgeSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// вызываем ф-ию создания пула соединений с БД, передаем структуру конфигурации
	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
	}
	defer db.Close()

	logger.Info("database connection pool established")

	// Declare an instance of the application struct
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	// Declare a HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Start the HTTP server.
	logger.Info("starting server", "addr", srv.Addr)
	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}

// openDB - ф-ия возвращает пул соединенией sql.DB.
func openDB(cfg config) (*sql.DB, error) {
	// создание пустого пула соединений
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Установить ограничение на кол-во открытых соединений
	// значение  <= 0, означает без ограничений
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Установить ограничение на кол-во неакт. соединений
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Установить ограничение на продолжительность неактивных соединений
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// создать контекст с таймаутом 5 с
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//С пом. PingContext установить новое соединение с БД передав context.
	//если соединение в т.5 сек не удалось, вернётся ошибка.
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}
	// sql.DB - пул соединений
	return db, nil

}
