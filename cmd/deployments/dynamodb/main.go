package main

import (
	"log"

	cfg "github.com/notifique/internal/config"
	ddb "github.com/notifique/internal/deployments"
	storage "github.com/notifique/internal/storage/dynamodb"
)

func main() {

	configurator, err := cfg.NewEnvConfig(".env")

	if err != nil {
		log.Fatal(err)
	}

	client, err := storage.NewDynamoDBClient(configurator)

	if err != nil {
		log.Fatalf("failed to create dynamo client - %v", err)
	}

	err = ddb.CreateTables(client)

	if err != nil {
		log.Fatalf("Failed to create tables - %v", err)
	}

	log.Print("Tables created!")
}
