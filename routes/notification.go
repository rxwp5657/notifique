package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/controllers"
)

func SetupRoutes(ns c.NotificationStorage) *gin.Engine {

	r := gin.Default()
	controller := c.NotificationController{Storage: ns}

	v0 := r.Group("/v0")
	{
		v0.GET("/users/notifications", controller.GetUserNotifications)
		v0.GET("/users/notifications/config", controller.GetUserConfig)
		v0.PUT("/users/notifications/:id/read", controller.SetReadStatus)
		v0.PUT("/users/notifications/opt-in", controller.OptIn)
		v0.PUT("/users/notifications/opt-out", controller.OptOut)

		v0.POST("/notifications", controller.CreateNotification)
	}

	return r
}
