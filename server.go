package main

import (
	"github.com/notifique/internal"
	"github.com/notifique/routes"
)

func main() {

	storage := internal.MakeInMemoryStorage()
	r := routes.SetupRoutes(&storage)

	r.Run() // listen and serve on 0.0.0.0:8080
}
