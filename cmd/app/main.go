package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/notifique/internal/publisher"
	storage "github.com/notifique/internal/storage/dynamodb"
	"github.com/notifique/routes"
)

func main() {

	dynamoBaseEndpoint := "http://localhost:8000"
	sqsBaseEndpoint := "http://localhost:8000"

	dynamoClient, err := storage.MakeDynamoDBClient(&dynamoBaseEndpoint)

	if err != nil {
		log.Fatalf(err.Error())
	}

	sqsClient, err := publisher.MakeSQSClient(&sqsBaseEndpoint)

	if err != nil {
		log.Fatalf(err.Error())
	}

	storage := storage.MakeDynamoDBStorage(dynamoClient)
	sqsPublisher := publisher.MakeSQSPublisher(sqsClient, publisher.SQSUrls{})

	r := gin.Default()

	routes.SetupNotificationRoutes(r, &storage, &sqsPublisher)
	routes.SetupDistributionListRoutes(r, &storage)
	routes.SetupUsersRoutes(r, &storage)

	r.Run() // listen and serve on 0.0.0.0:8080
}
