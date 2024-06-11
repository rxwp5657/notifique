package main

import (
	"log"

	di "github.com/notifique/dependency_injection"
	"github.com/notifique/internal/config"
)

func main() {

	loader, err := config.MakeEnvConfig(".env")

	if err != nil {
		log.Fatal(err)
	}

	r, err := di.InjectDynamoSQSEngine(loader)

	if err != nil {
		log.Fatalf("failed to create engine - %v", err)
	}

	r.Run() // listen and serve on 0.0.0.0:8080
}
