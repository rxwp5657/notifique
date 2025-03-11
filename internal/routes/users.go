package routes

import (
	"errors"

	"github.com/gin-gonic/gin"

	c "github.com/notifique/internal/controllers"
)

func SetupUsersRoutes(r *gin.Engine, version string, controller *c.UserController) error {

	if controller == nil {
		return errors.New("users controller is nil")
	}

	g := r.Group(version)
	{
		g.GET("/users/me/notifications", controller.GetUserNotifications)
		g.GET("/users/me/notifications/live", controller.GetLiveUserNotifications)
		g.PATCH("/users/me/notifications/:id", controller.SetReadStatus)

		g.GET("/users/me/notifications/config", controller.GetUserConfig)
		g.PUT("/users/me/notifications/config", controller.UpdateUserConfig)
	}

	return nil
}
