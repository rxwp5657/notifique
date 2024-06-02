package main

import (
	"log"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	p "github.com/notifique/deployments/postgres"
)

func main() {

	url := "postgres://postgres:postgres@localhost:5432/notifique?sslmode=disable"

	err := p.RunMigrations(url)

	if err != nil {
		log.Fatalf("failed to execute migrations - %v", err)
	}

	log.Println("migrations executed!")
}
