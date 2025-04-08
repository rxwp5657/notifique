package internal

import (
	"errors"
	"regexp"
)

func GetBasePath(path, basePath string) (string, error) {

	r, _ := regexp.Compile(basePath)
	path = r.FindString(path)

	if path == "" {
		return "", errors.New("invalid path")
	}

	return path, nil
}
