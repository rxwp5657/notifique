package main

import (
	"log"

	deployments "github.com/notifique/deployments/sqs"
	"github.com/notifique/internal/publisher"
)

func main() {

	url := "http://localhost:4566"

	client, err := publisher.MakeSQSClient(&url)

	if err != nil {
		log.Fatalf("failed to make sqs client - %v", err)
	}

	_, err = deployments.MakeQueues(client)

	if err != nil {
		log.Fatalf("failed to make sqs queues - %v", err)
	}

	log.Println("queues created!")
}
