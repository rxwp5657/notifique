package main

import (
	"log"

	ddb "github.com/notifique/deployments/dynamodb"
	storage "github.com/notifique/internal/storage/dynamodb"
)

func main() {

	url := "http://localhost:8000"
	client, err := storage.MakeDynamoDBClient(&url)

	if err != nil {
		log.Fatalf("failed to create dynamo client - %v", err)
	}

	err = ddb.CreateTables(client)

	if err != nil {
		log.Fatalf("Failed to create tables - %v", err)
	}

	log.Print("Tables created!")
}
