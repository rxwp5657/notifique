package postgresresgistry

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
)

func marshallKey[T any](key T) (string, error) {
	jsonMarshal, err := json.Marshal(key)

	if err != nil {
		return "", fmt.Errorf("failed to json marshall key - %w", err)
	}

	base64Encoded := b64.StdEncoding.EncodeToString(jsonMarshal)

	return base64Encoded, nil
}

func unmarsallKey[T any](marshalled string, key *T) error {
	base64Decoded, err := b64.StdEncoding.DecodeString(marshalled)

	if err != nil {
		return fmt.Errorf("failed to decode base64 key - %w", err)
	}

	err = json.Unmarshal(base64Decoded, key)

	if err != nil {
		return fmt.Errorf("failed to json unmarshall key - %w", err)
	}

	return nil
}
