package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/controllers"
)

func SetupNotificationRoutes(r *gin.Engine, ns c.NotificationStorage) {

	controller := c.NotificationController{Storage: ns}

	v0 := r.Group("/v0")
	{
		v0.GET("/notifications", controller.GetNotifications)
		v0.POST("/notifications", controller.CreateNotification)
		v0.PUT("/notifications/:id/read", controller.SetReadStatus)
		v0.PUT("/notifications/opt-in", controller.OptIn)
		v0.PUT("/notifications/opt-out", controller.OptOut)
	}
}
