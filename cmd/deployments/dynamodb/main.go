package main

import (
	"log"

	ddb "github.com/notifique/deployments/dynamodb"
	storage "github.com/notifique/internal/storage/dynamodb"
)

func main() {

	url := "http://localhost:8000"
	endpoint := (storage.DynamoEndpoint)(url)
	client, err := storage.MakeDynamoDBClient(&endpoint)

	if err != nil {
		log.Fatalf("failed to create dynamo client - %v", err)
	}

	err = ddb.CreateTables(client)

	if err != nil {
		log.Fatalf("Failed to create tables - %v", err)
	}

	log.Print("Tables created!")
}
