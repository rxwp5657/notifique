package internal

import "fmt"

type NotificationNotFound struct {
	NotificationId string
}

func (e NotificationNotFound) Error() string {
	return fmt.Sprintf("Notification %v not found", e.NotificationId)
}
