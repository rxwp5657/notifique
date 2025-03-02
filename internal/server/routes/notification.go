package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/internal/server/controllers"
)

func SetupNotificationRoutes(r *gin.Engine, version string, ns c.NotificationRegistry, p c.NotificationPublisher) {

	controller := c.NotificationController{Registry: ns, Publisher: p}

	g := r.Group(version)
	{
		g.POST("/notifications", controller.CreateNotification)
		g.DELETE("/notifications/:id", controller.DeleteNotification)
	}
}
