package data

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

// тип пароля содержит незашифрованную и хешированную версию пароля пользователя.
// Поле незашифрованного пароля является *указателем* на строку,
// поэтому мы можем отличить ситуацию, когда незашифрованный пароль вообще отсутствует в
// структуре, от ситуации, когда незашифрованный пароль представляет собой пустую строку ""
type password struct {
	plaintext *string
	hash      []byte
}

// Set() вычисляет хэш bcrypt для незашифрованного пароля и сохраняет в структуре как хэш, так и незашифрованный пароль.
func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}
	p.plaintext = &plaintextPassword
	p.hash = hash
	return nil
}

// Matches() метод проверяет, соответствует ли предоставленный текстовый пароль хэшированному паролю хранящемуся в структуре,
// и возвращает true, если он совпадает, и false  в противном случае.
func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}
