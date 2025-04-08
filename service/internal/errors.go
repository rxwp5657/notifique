package internal

import "fmt"

type DistributionListAlreadyExists struct {
	Name string
}

type InvalidNotificationStatus struct {
	Id     string
	Status string
}

type EntityNotFound struct {
	Id   string
	Type string
}

func (e EntityNotFound) Error() string {
	return fmt.Sprintf("entity %v of type %v not found", e.Id, e.Type)
}

func (e DistributionListAlreadyExists) Error() string {
	return fmt.Sprintf("distribution list %v already exists", e.Name)
}

func (e InvalidNotificationStatus) Error() string {
	return fmt.Sprintf("notification %v has status %v", e.Id, e.Status)
}
