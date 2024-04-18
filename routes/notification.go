package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	c "github.com/notifique/controllers"
	"github.com/notifique/internal"
)

func SetupRoutes(ns c.NotificationStorage) *gin.Engine {

	r := gin.Default()

	controller := c.NotificationController{Storage: ns}

	v0 := r.Group("/v0")
	{
		v0.GET("/notifications", controller.GetUserNotifications)
		v0.GET("/notifications/config", controller.GetUserConfig)
		v0.PATCH("/notifications/:id", controller.SetReadStatus)
		v0.PATCH("/notifications/config", controller.UpdateUserConfig)

		v0.POST("/notifications", controller.CreateNotification)
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("future", internal.FutureValidator)
	}

	return r
}
