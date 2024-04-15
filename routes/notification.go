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
		v0.GET("/notifications", controller.GetUserNotifications)
		v0.GET("/notifications/config", controller.GetUserConfig)
		v0.PATCH("/notifications/:id", controller.SetReadStatus)
		v0.PATCH("/notifications/config", controller.UpdateUserConfig)

		v0.POST("/notifications", controller.CreateNotification)
	}

	return r
}
