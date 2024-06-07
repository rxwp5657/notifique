package main

import (
	"log"

	di "github.com/notifique/dependency_injection"
	"github.com/notifique/internal/publisher"
)

func main() {
	r, err := di.InjectDynamoSQSEngine(nil, nil, publisher.SQSEndpoints{})

	if err != nil {
		log.Fatalf("failed to create engine - %v", err)
	}

	r.Run() // listen and serve on 0.0.0.0:8080
}
