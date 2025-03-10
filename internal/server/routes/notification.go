package routes

import (
	"errors"

	"github.com/gin-gonic/gin"

	c "github.com/notifique/internal/server/controllers"
)

func SetupNotificationRoutes(r *gin.Engine, version string, controller *c.NotificationController) error {

	if controller == nil {
		return errors.New("notifications controller is nil")
	}

	g := r.Group(version)
	{
		g.GET("/notifications", controller.GetNotifications)
		g.GET("/notifications/:id", controller.GetNotification)
		g.POST("/notifications", controller.CreateNotification)
		g.POST("/notifications/:id/cancel", controller.CancelDelivery)
		g.DELETE("/notifications/:id", controller.DeleteNotification)
	}

	return nil
}
