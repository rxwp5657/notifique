package main

import (
	"github.com/gin-gonic/gin"
	storage "github.com/notifique/internal/storage/dynamodb"
	"github.com/notifique/routes"
)

func main() {

	storage := storage.MakeDynamoDBStorage()
	r := gin.Default()

	routes.SetupNotificationRoutes(r, &storage)
	routes.SetupDistributionListRoutes(r, &storage)
	routes.SetupUsersRoutes(r, &storage)

	r.Run() // listen and serve on 0.0.0.0:8080
}
