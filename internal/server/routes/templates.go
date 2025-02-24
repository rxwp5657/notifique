package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/notifique/internal/server"
	c "github.com/notifique/internal/server/controllers"
)

func SetupNotificationTemplateRoutes(r *gin.Engine, version string, ntr c.NotificationTemplateRegistry) {

	controller := c.NotificationTemplateController{Registry: ntr}

	g := r.Group(version)
	{
		g.POST("/notifications/templates", controller.CreateNotificationTemplate)
		g.GET("/notifications/templates", controller.GetNotifications)
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("unique_var_name", server.UniqueTemplateVarValidator)
	}
}
