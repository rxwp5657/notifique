package internal

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	r "regexp"

	"github.com/go-playground/validator/v10"
	"github.com/notifique/internal/dto"
)

type TemplateVariableValidator func(string) error

const TemplateVariableNameSeparator = "~"

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

func validateTemplateVar(tv dto.TemplateVariable, suppliedVar dto.TemplateVariableContents) error {

	validateRegex := func(val string, pattern *string) error {

		if pattern == nil {
			return nil
		}

		match, err := r.MatchString(*pattern, val)

		if err != nil {
			return fmt.Errorf("%s regex is invalid", *pattern)
		}

		if !match {
			return fmt.Errorf("%s failed regex validation %s", val, *pattern)
		}

		return nil
	}

	validateNumber := func(val string) error {
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			return fmt.Errorf("%s is not a number", val)
		}

		return nil
	}

	validateDate := func(val string) error {
		_, err := time.Parse(time.DateOnly, val)

		if err != nil {
			return fmt.Errorf("%s is not a valid date (YYYY-MM-DD)", val)
		}

		return nil
	}

	validateDateTime := func(val string) error {
		_, err := time.Parse(time.RFC3339, val)

		if err != nil {
			return fmt.Errorf("%s is not a valid RFC3339 datetime", val)
		}

		return nil
	}

	validateString := func(val string) error {
		return nil
	}

	validate := func(val string, validator TemplateVariableValidator) error {

		if err := validator(val); err != nil {
			return err
		}

		if tv.Validation != nil {
			err := validateRegex(val, tv.Validation)
			return err
		}

		return nil
	}

	validators := map[string]TemplateVariableValidator{
		string(dto.Number):   validateNumber,
		string(dto.Date):     validateDate,
		string(dto.DateTime): validateDateTime,
		string(dto.String):   validateString,
	}

	v := suppliedVar.Value
	validator, ok := validators[tv.Type]

	if !ok {
		return fmt.Errorf("validator not found for type %s", tv.Type)
	}

	return validate(v, validator)
}

func ValidateTemplateVars(templateVars []dto.TemplateVariable, suppliedVars []dto.TemplateVariableContents) error {

	templateVarsMap := make(map[string]dto.TemplateVariable, len(templateVars))
	requiredTemplateVars := []string{}

	for _, tv := range templateVars {
		templateVarsMap[tv.Name] = tv
		if tv.Required {
			requiredTemplateVars = append(requiredTemplateVars, tv.Name)
		}
	}

	errorArr := []error{}
	validatedVariables := map[string]struct{}{}

	for _, sv := range suppliedVars {
		tv, ok := templateVarsMap[sv.Name]

		if !ok {
			errorArr = append(errorArr, fmt.Errorf("%s is not a template variable", sv.Name))
			continue
		}

		errorArr = append(errorArr, validateTemplateVar(tv, sv))
		validatedVariables[sv.Name] = struct{}{}
	}

	for _, v := range requiredTemplateVars {
		_, validated := validatedVariables[v]

		if !validated {
			errorArr = append(errorArr, fmt.Errorf("template variable %s not found", v))
		}
	}

	return errors.Join(errorArr...)
}

var TemplateNameValidator validator.Func = func(fl validator.FieldLevel) bool {
	str, ok := fl.Field().Interface().(string)

	if !ok {
		return false
	}

	return !strings.Contains(str, TemplateVariableNameSeparator)
}
