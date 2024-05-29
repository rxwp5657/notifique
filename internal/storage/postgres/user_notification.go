package storage

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/notifique/dto"
)

type userNotification struct {
	Id        string     `db:"id"`
	Title     string     `db:"title"`
	Contents  string     `db:"contents"`
	CreatedAt time.Time  `db:"created_at"`
	ImageUrl  *string    `db:"image_url"`
	ReadAt    *time.Time `db:"read_at"`
	Topic     string     `db:"topic"`
}

type userNotificationKey struct {
	Id        string `json:"id"`
	UserId    string `json:"userId"`
	CreatedAt string `json:"createdAt"`
}

func (k *userNotificationKey) marshal() (string, error) {
	jsonMarshal, err := json.Marshal(k)

	if err != nil {
		return "", fmt.Errorf("failed to json marshall key - %w", err)
	}

	base64Encoded := b64.StdEncoding.EncodeToString(jsonMarshal)

	return base64Encoded, nil
}

func (n *userNotification) toDTO() dto.UserNotification {
	var readAt *string = nil

	if n.ReadAt != nil {
		parsed := n.ReadAt.Format(time.RFC3339Nano)
		readAt = &parsed
	}

	notification := dto.UserNotification{
		Id:        n.Id,
		Title:     n.Title,
		Contents:  n.Contents,
		CreatedAt: n.CreatedAt.Format(time.RFC3339Nano),
		Image:     n.ImageUrl,
		ReadAt:    readAt,
		Topic:     n.Topic,
	}

	return notification
}

func unmarshallNotificationKey(key string) (*userNotificationKey, error) {
	base64Decoded, err := b64.StdEncoding.DecodeString(key)

	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 key - %w", err)
	}

	var ukey userNotificationKey

	err = json.Unmarshal(base64Decoded, &ukey)

	if err != nil {
		return nil, fmt.Errorf("failed to json unmarshall key - %w", err)
	}

	return &ukey, nil
}

const GET_USER_NOTIFICATIONS = `
SELECT
	id,
	title,
	contents,
	created_at,
	image_url,
	read_at,
	topic
FROM
	user_notifications
%s
ORDER BY
	created_at DESC
LIMIT
	@limit;
`

const INSERT_USER_NOTIFICATION = `
INSERT INTO user_notifications(
	id,
	user_id,
	title,
	contents,
	created_at,
	read_at,
	image_url,
	topic
) VALUES (
	@id,
	@userId,
	@title,
	@contents,
	@createdAt,
	@readAt,
	@imageUrl,
	@topic
);
`

const DELETE_USER_NOTIFICATION = `
DELETE FROM user_notifications WHERE id = @id;
`
