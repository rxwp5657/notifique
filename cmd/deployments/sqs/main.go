package main

import (
	"log"

	deployments "github.com/notifique/deployments/sqs"
	"github.com/notifique/internal/publisher"
)

func main() {

	url := "http://localhost:4566"

	baseUrl := (publisher.SQSEndpoint)(url)
	client, err := publisher.MakeSQSClient(&baseUrl)

	if err != nil {
		log.Fatalf("failed to make sqs client - %v", err)
	}

	_, err = deployments.MakeQueues(client)

	if err != nil {
		log.Fatalf("failed to make sqs queues - %v", err)
	}

	log.Println("queues created!")
}
