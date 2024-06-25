package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/controllers"
)

func SetupNotificationRoutes(r *gin.Engine, ns c.NotificationStorage, p c.NotificationPublisher) {

	controller := c.NotificationController{Storage: ns, Publisher: p}

	g := r.Group("/v0")
	{
		g.POST("/notifications", controller.CreateNotification)
	}
}
