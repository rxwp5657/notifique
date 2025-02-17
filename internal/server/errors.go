package server

import "fmt"

type NotificationNotFound struct {
	NotificationId string
}

type DistributionListAlreadyExists struct {
	Name string
}

type DistributionListNotFound struct {
	Name string
}

func (e NotificationNotFound) Error() string {
	return fmt.Sprintf("Notification %v not found", e.NotificationId)
}

func (e DistributionListAlreadyExists) Error() string {
	return fmt.Sprintf("Distribution list %v already exists", e.Name)
}

func (e DistributionListNotFound) Error() string {
	return fmt.Sprintf("Distribution list %v not found", e.Name)
}
