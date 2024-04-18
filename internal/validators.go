package internal

import (
	"time"

	"github.com/go-playground/validator/v10"
)

var FutureValidator validator.Func = func(fl validator.FieldLevel) bool {
	dateStr, ok := fl.Field().Interface().(string)

	if !ok {
		return false
	}

	dateTime, _ := time.Parse(time.RFC3339, dateStr)

	return !dateTime.Before(time.Now())
}
