package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/controllers"
)

func SetupNotificationRoutes(r *gin.Engine, ns c.NotificationStorage) {

	controller := c.NotificationController{Storage: ns}

	v0 := r.Group("/v0")
	{
		v0.POST("/notifications", controller.CreateNotification)
	}
}
