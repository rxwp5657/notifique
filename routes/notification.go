package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/controllers"
)

func SetupNotificationRoutes(r *gin.Engine, ns c.NotificationStorage, p c.NotificationPublisher) {

	controller := c.NotificationController{Storage: ns, Publisher: p}

	v0 := r.Group("/v0")
	{
		v0.POST("/notifications", controller.CreateNotification)
	}
}
