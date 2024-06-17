package main

import (
	"log"

	ddb "github.com/notifique/deployments/dynamodb"
	cfg "github.com/notifique/internal/config"
	storage "github.com/notifique/internal/storage/dynamodb"
)

func main() {

	configurator, err := cfg.MakeEnvConfig(".env")

	if err != nil {
		log.Fatal(err)
	}

	client, err := storage.MakeDynamoDBClient(configurator)

	if err != nil {
		log.Fatalf("failed to create dynamo client - %v", err)
	}

	err = ddb.CreateTables(client)

	if err != nil {
		log.Fatalf("Failed to create tables - %v", err)
	}

	log.Print("Tables created!")
}
