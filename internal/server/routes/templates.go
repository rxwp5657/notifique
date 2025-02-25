package routes

import (
	"github.com/gin-gonic/gin"
	c "github.com/notifique/internal/server/controllers"
)

func SetupNotificationTemplateRoutes(r *gin.Engine, version string, ntr c.NotificationTemplateRegistry) {

	controller := c.NotificationTemplateController{Registry: ntr}

	g := r.Group(version)
	{
		g.POST("/notifications/templates", controller.CreateNotificationTemplate)
		g.GET("/notifications/templates", controller.GetNotifications)
		g.GET("/notifications/templates/:id", controller.GetNotification)
	}
}
