package main

import (
	"github.com/gin-gonic/gin"
	"github.com/notifique/internal"
	"github.com/notifique/routes"
)

func main() {

	storage := internal.MakeInMemoryStorage()
	r := gin.Default()

	routes.SetupNotificationRoutes(r, &storage)
	routes.SetupDistributionListRoutes(r, &storage)

	r.Run() // listen and serve on 0.0.0.0:8080
}
