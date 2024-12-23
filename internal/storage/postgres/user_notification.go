package storage

import (
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
	Id     string `json:"id"`
	UserId string `json:"userId"`
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

const GetUserNotifications = `
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

const UpdateReadAt = `
UPDATE
	user_notifications
SET
	read_at = NOW()
WHERE
	id = @id AND
	user_id = @userId
RETURNING
	id;
`
