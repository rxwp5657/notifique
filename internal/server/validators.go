package server

import (
	"time"

	r "regexp"

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

var DLNameValidator validator.Func = func(fl validator.FieldLevel) bool {
	name, ok := fl.Field().Interface().(string)

	if !ok {
		return false
	}

	match, err := r.MatchString("^[A-Za-z0-9$#@-_]+$", name)

	if err != nil {
		return false
	}

	return match
}
