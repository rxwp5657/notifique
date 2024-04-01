package main

import (
	"github.com/gin-gonic/gin"

	"github.com/notifique/internal"
	"github.com/notifique/routes"
)

func main() {
	r := gin.Default()

	storage := internal.MakeInMemoryStorage()
	routes.SetupNotificationRoutes(r, &storage)

	r.Run() // listen and serve on 0.0.0.0:8080
}
