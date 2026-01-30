package data

import (
	"database/sql"
	"errors"
)

// ошибка, в случае отсутствия данных в БД
var (
	ErrRecordNotFound = errors.New("record not found")
)

// Структура Models - обёртка для других моделей
type Models struct {
	Movies MovieModel
}

// метод возвращает структуру инициализированную Models
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
	}
}
