package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ошибка при невозможности парсинга runtime
var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

// Объявим собственный тип.
type Runtime int32

// MarshalJSON - вернет значение в формате JSON "? mins"
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)

	//заключим строку в двойные кавычки, для корректного json
	quotedJSONValue := strconv.Quote(jsonValue)

	// конвертация в byte slice
	return []byte(quotedJSONValue), nil
}

// метод удовлетворяет json.Unmarshaler. Исп-ся указатель Runtime, иначе будем изменять только копию.
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// Ожидаемя строка  "? mins".Удалим кавычки.
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// выделим число
	parts := strings.Split(unquotedJSONValue, " ")

	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}
	*r = Runtime(i)
	return nil
}
