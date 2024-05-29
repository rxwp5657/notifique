package main

import (
	"log"

	"github.com/gin-gonic/gin"
	storage "github.com/notifique/internal/storage/dynamodb"
	"github.com/notifique/routes"
)

func main() {

	baseEndpoint := "http://localhost:8000"
	client, err := storage.MakeClient(&baseEndpoint)

	if err != nil {
		log.Fatalf(err.Error())
	}

	storage := storage.MakeDynamoDBStorage(client)
	r := gin.Default()

	routes.SetupNotificationRoutes(r, &storage)
	routes.SetupDistributionListRoutes(r, &storage)
	routes.SetupUsersRoutes(r, &storage)

	r.Run() // listen and serve on 0.0.0.0:8080
}
