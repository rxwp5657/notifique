package main

import (
	"log"

	di "github.com/notifique/dependency_injection"
	cfg "github.com/notifique/internal/config"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	p "github.com/notifique/deployments/postgres"
)

func main() {

	loader, err := cfg.MakeEnvConfig(".env")

	if err != nil {
		log.Fatal(err)
	}

	url, ok := loader.GetConfigValue(di.POSTGRES_URL)

	if !ok {
		log.Fatalf("postgres url %s is not set", di.POSTGRES_URL)
	}

	err = p.RunMigrations(url)

	if err != nil {
		log.Fatalf("failed to execute migrations - %v", err)
	}

	log.Println("migrations executed!")
}
