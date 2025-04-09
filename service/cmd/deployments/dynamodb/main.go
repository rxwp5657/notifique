package main

import (
	"log"

	cfg "github.com/notifique/service/internal/config"
	ddb "github.com/notifique/service/pkg/deployments"
	"github.com/notifique/shared/clients"
)

func main() {

	env := "./config/local.env"
	configurator, err := cfg.NewEnvConfig(&env)

	if err != nil {
		log.Fatal(err)
	}

	client, err := clients.NewDynamoDBClient(configurator)

	if err != nil {
		log.Fatalf("failed to create dynamo client - %v", err)
	}

	err = ddb.CreateTables(client)

	if err != nil {
		log.Fatalf("Failed to create tables - %v", err)
	}

	log.Print("Tables created!")
}
