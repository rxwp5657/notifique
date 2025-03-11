package routes

import (
	"errors"

	"github.com/gin-gonic/gin"
	c "github.com/notifique/internal/controllers"
)

func SetupNotificationTemplateRoutes(r *gin.Engine, version string, controller *c.NotificationTemplateController) error {

	if controller == nil {
		return errors.New("distribution lists controller is nil")
	}

	g := r.Group(version)
	{
		g.POST("/notifications/templates", controller.CreateNotificationTemplate)
		g.GET("/notifications/templates", controller.GetTemplates)
		g.GET("/notifications/templates/:id", controller.GetTemplateDetails)
		g.DELETE("/notifications/templates/:id", controller.DeleteTemplate)
	}

	return nil
}
