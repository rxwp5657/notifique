package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/controllers"
)

func SetupNotificationRoutes(r *gin.Engine, version string, ns c.NotificationStorage, p c.NotificationPublisher) {

	controller := c.NotificationController{Storage: ns, Publisher: p}

	g := r.Group(version)
	{
		g.POST("/notifications", controller.CreateNotification)
	}
}
