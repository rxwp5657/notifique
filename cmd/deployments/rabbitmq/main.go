package main

import (
	"log"

	di "github.com/notifique/internal/di"
)

func main() {

	env := "./config/local.env"
	deployer, cleanup, err := di.InjectRabbitMQPriorityDeployer(&env)

	if err != nil {
		log.Fatal(err)
	}

	defer cleanup()

	err = deployer.Deploy()

	if err != nil {
		log.Fatalf("failed to deploy queues - %v", err)
	}

	log.Print("queues deployed")
}
