package main

import (
	"log"

	di "github.com/notifique/dependency_injection"
	cfg "github.com/notifique/internal/config"
)

func main() {

	loader, err := cfg.MakeEnvConfig(".env")

	if err != nil {
		log.Fatal(err)
	}

	deployment, err := di.InjectRabbitMQPriorityQueueDeployment(loader)

	if err != nil {
		log.Fatalf("failed to create deployment - %v", err)
	}

	defer deployment.Client.Close()

	err = deployment.Deploy()

	if err != nil {
		log.Fatalf("failed to deploy queues - %v", err)
	}

	log.Print("queues deployed")
}
