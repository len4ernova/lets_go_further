package data

import (
	"database/sql"
	"time"

	"github.com/len4ernova/lets_go_further/internal/validator"
)

type Movie struct {
	ID        int64     `json:"id"`               // Unique integer ID for the movie
	CreatedAt time.Time `json:"-"`                // Timestamp for when the movie is added to our database
	Title     string    `json:"title"`            // Movie title
	Year      int32     `json:"year,omitzero"`    // Movie release year
	Runtime   Runtime   `json:"runtime,omitzero"` // Movie runtime (in minutes)
	Genres    []string  `json:"genres,omitzero"`  // Slice of genres for the movie (romance, comedy, etc.)
	Version   int32     `json:"version"`          // The version number starts at 1 and will be incremented each
	// time the movie information is updated
}

// определим структуру, являющуюся оберткой для БД
type MovieModel struct {
	DB *sql.DB
}

// ValidateMovie - проверка валидности данных структуры Movie
func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must conain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")
	//проверка уникальности жанров
	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")

}

// placeholder - метод создающий запись в БД.
func (m MovieModel) Insert(movie *Movie) error {
	return nil
}

// placeholder - метод получение конкретной записи из БД.
func (m MovieModel) Get(id int64) (*Movie, error) {
	return nil, nil
}

// placeholder - метод обновляет конкретную запись в БД.
func (m MovieModel) update(movie *Movie) error {
	return nil
}

// placeholder - метод удаляет определнную запись в БД.
func (m MovieModel) Delete(id int64) error {
	return nil
}
