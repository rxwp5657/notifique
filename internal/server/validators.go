package server

import (
	"time"

	r "regexp"

	"github.com/go-playground/validator/v10"
	"github.com/notifique/internal/server/dto"
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

var UniqueTemplateVarValidator validator.Func = func(fl validator.FieldLevel) bool {
	templateVariables, ok := fl.Field().Interface().([]dto.TemplateVariable)

	if !ok {
		return false
	}

	variableNames := map[string]struct{}{}

	for _, v := range templateVariables {
		if _, ok := variableNames[v.Name]; ok {
			return false
		}

		variableNames[v.Name] = struct{}{}
	}

	return true
}
