package server

import "fmt"

type DistributionListAlreadyExists struct {
	Name string
}

type EntityNotFound struct {
	Id   string
	Type string
}

func (e EntityNotFound) Error() string {
	return fmt.Sprintf("Entity %v of type %v not found", e.Id, e.Type)
}

func (e DistributionListAlreadyExists) Error() string {
	return fmt.Sprintf("Distribution list %v already exists", e.Name)
}
