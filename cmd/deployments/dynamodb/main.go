package main

import (
	"log"

	di "github.com/notifique/dependency_injection"
	ddb "github.com/notifique/deployments/dynamodb"
	cfg "github.com/notifique/internal/config"
	storage "github.com/notifique/internal/storage/dynamodb"
)

func main() {

	loader, err := cfg.MakeEnvConfig(".env")

	if err != nil {
		log.Fatal(err)
	}

	var url *string = nil

	if be, ok := loader.GetConfigValue(di.DYNAMO_BASE_ENDPOINT); ok {
		url = &be
	}

	cfg := storage.DynamoClientConfig{BaseEndpoint: url}

	client, err := storage.MakeDynamoDBClient(cfg)

	if err != nil {
		log.Fatalf("failed to create dynamo client - %v", err)
	}

	err = ddb.CreateTables(client)

	if err != nil {
		log.Fatalf("Failed to create tables - %v", err)
	}

	log.Print("Tables created!")
}
