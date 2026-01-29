package validator

import (
	"regexp"
	"slices"
)

// источник регулярного выражения для проверки email:
//
//	https://html.spec.whatwg.org/#valid-e-mail-address
var (
	EmailRF = "^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"
)

// объявим тип Validator, содержащий map с ошибками
type Validator struct {
	Errors map[string]string
}

// New - вспомогательня ф-ия содержащая пустой словарь ошибок.
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Valid - возвращает true, если нет ошибок.
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError - добавить ошибку в map (если в map нет такого ключа)
func (v *Validator) AddError(key, message string) {
	if _, exist := v.Errors[key]; !exist {
		v.Errors[key] = message
	}
}

// Check - добавляет сообщение об ошибке, если валидация не прошла.
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// PermittedValue - универсальная ф-ия, возвращает true, если значение находится в списке допустимых значений
func PermittedValue[T comparable](value T, permitedValues ...T) bool {
	return slices.Contains(permitedValues, value)
}

// Matches - возвращает true, если строка соотв-ет определенному шаблону.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// возвращает true, если значения в срезе уникальны
func Unique[T comparable](values []T) bool {
	uniqueValues := make(map[T]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}
