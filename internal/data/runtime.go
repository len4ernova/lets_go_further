package data

import (
	"fmt"
	"strconv"
)

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
