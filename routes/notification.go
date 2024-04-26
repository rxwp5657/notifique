package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	c "github.com/notifique/controllers"
	"github.com/notifique/internal"
)

func SetupNotificationRoutes(r *gin.Engine, ns c.NotificationStorage) {

	controller := c.NotificationController{Storage: ns}

	v0 := r.Group("/v0")
	{
		v0.GET("/users/me/notifications", controller.GetUserNotifications)
		v0.POST("/users/:id/notifications", controller.CreateUserNotification)
		v0.PATCH("/users/me/notifications/:id", controller.SetReadStatus)

		v0.GET("/users/me/notifications/config", controller.GetUserConfig)
		v0.PATCH("/users/me/notifications/config", controller.UpdateUserConfig)

		v0.POST("/notifications", controller.CreateNotification)
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("future", internal.FutureValidator)
	}
}
