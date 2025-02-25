package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/internal/server/controllers"
)

func SetupUsersRoutes(r *gin.Engine, version string, us c.UserRegistry, bk c.UserNotificationBroker) {

	controller := c.UserController{
		Registry: us,
		Broker:   bk,
	}

	g := r.Group(version)
	{
		g.GET("/users/me/notifications", controller.GetUserNotifications)
		g.GET("/users/me/notifications/live", controller.GetLiveUserNotifications)
		g.PATCH("/users/me/notifications/:id", controller.SetReadStatus)

		g.GET("/users/me/notifications/config", controller.GetUserConfig)
		g.PUT("/users/me/notifications/config", controller.UpdateUserConfig)
	}
}
