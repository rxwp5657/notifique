package main

import (
	"log"

	di "github.com/notifique/internal/di"
)

func main() {

	r, close, err := di.InjectPgPriorityRabbitMQ(nil)

	if err != nil {
		log.Fatalf("failed to create engine - %v", err)
	}

	defer close()

	r.Run() // listen and serve on 0.0.0.0:8080
}
