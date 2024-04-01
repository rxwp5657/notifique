package internal

import "fmt"

type NotificationNotFound struct {
	NotificationId string
}

type RecipientNotFound struct {
	NotificationId string
	UserId         string
}

func (e NotificationNotFound) Error() string {
	return fmt.Sprintf("Notification %v not found", e.NotificationId)
}

func (e RecipientNotFound) Error() string {
	msg := "User %v doesn't have the notification with id %v"
	return fmt.Sprintf(msg, e.UserId, e.NotificationId)
}
