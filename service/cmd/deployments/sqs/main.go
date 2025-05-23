package main

import (
	"log"

	di "github.com/notifique/service/internal/di"
)

func main() {

	env := "./config/local.env"
	deployer, cleanup, err := di.InjectSQSPriorityDeployer(&env)

	if err != nil {
		log.Fatalf("failed to create deployment - %v", err)
	}

	defer cleanup()

	_, err = deployer.Deploy()

	if err != nil {
		log.Fatalf("failed to deploy queues - %v", err)
	}

	log.Println("queues created!")
}
