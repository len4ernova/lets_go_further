package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/len4ernova/lets_go_further/internal/data"
	"github.com/len4ernova/lets_go_further/internal/mailer"
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
	//  структура-ограничитель, содержит поля для значений количества запросов в секунду и пиковой нагрузки,
	// а также логическое поле, с помощью которх можем включать и выключать ограничение скорости
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	// конфигурация smtp-сервера
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	// cors
	cors struct {
		trustedOrigins []string
	}
}

// an application struct to hold the dependencies for our HTTP handlers, helpers, and middleware
type application struct {
	config config
	logger *slog.Logger
	models data.Models
	mailer *mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg config

	// Read the value of the port and env command-line flags into the config struct.
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	//flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://user://pass@localhost/greenlight", "PostgeSQL DSN")
	// flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://user:pass@localhost/greenlight?sslmode=disable", "PostgeSQL DSN")
	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgeSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "a7420fc0883489", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "e75ffd0a3aa5ec", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// вызываем ф-ию создания пула соединений с БД, передаем структуру конфигурации
	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
	}
	defer db.Close()

	logger.Info("database connection pool established")

	// инициализируем экземпляр Mailer параметтрами командной строки
	mailer, err := mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	// опубликуем номер версии в expvar handler
	expvar.NewString("version").Set(version)

	// опубликуем кол-во активных горутин
	expvar.Publish("gorutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	// опубликуем статистику пула соединений с БД
	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))

	// опубликуем текущее Unix timestam
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	// Declare an instance of the application struct
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer,
	}

	// call a HTTP server
	err = app.server()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

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
