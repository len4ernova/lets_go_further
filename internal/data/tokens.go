package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"time"

	"github.com/len4ernova/lets_go_further/internal/validator"
)

// в файле хранится вся логика связанная с токенами

// константы - области видимости токенов
const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
	ScopePasswordReset  = "password-reset"
)

// данные по токену
type Token struct {
	Plaintext string    `json:"token"` // исх.токен
	Hash      []byte    `json:"-"`     // хеш токена
	UserID    int64     `json:"-"`
	Expire    time.Time `json:"expiry"` // время жизни
	Scope     string    `json:"-"`      //  область применения токена
}

// generateToken - сформировать токен.
func generateToken(UserID int64, ttl time.Duration, scope string) *Token {
	token := &Token{
		Plaintext: rand.Text(),
		UserID:    UserID,
		Expire:    time.Now().Add(ttl),
		Scope:     scope,
	}

	// сгенерируем хеш для токена
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:] // [:] преобразовывает в слайс (для удобства)

	return token
}

// валидация
// ValidateTokenPlaintext - проверка, представленный токен имеет длину 26 байт
func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

// TokenModel
type TokenModel struct {
	DB *sql.DB
}

// New - добавит данные в таблицу.
func (m TokenModel) New(userId int64, ttl time.Duration, scope string) (*Token, error) {
	token := generateToken(userId, ttl, scope)
	err := m.Insert(token)
	return token, err
}

// Insert - добавить токен в таблицу.
func (m TokenModel) Insert(token *Token) error {
	query := `
	INSERT INTO tokens (hash, user_id, expire, scope)
	VALUES ($1, $2, $3, $4)`

	args := []any{token.Hash, token.UserID, token.Expire, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

// DeleteToken - удалит токен определеннного пользователя.
func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
		DELETE FROM tokens 
		WHERE scope = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, scope, userID)
	return err
}
