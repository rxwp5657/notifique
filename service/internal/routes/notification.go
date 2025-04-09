package routes

import (
	c "github.com/notifique/service/internal/controllers"
	"github.com/notifique/shared/auth"
)

type notificationsRoutesCfg struct {
	routeGroupCfg
	Controller *c.NotificationController
}

func SetupNotificationRoutes(cfg notificationsRoutesCfg) error {

	g := cfg.Engine.Group(cfg.Version)
	{
		g.GET("/notifications",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.GetNotifications)

		g.GET("/notifications/:id",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.GetNotification)

		g.POST("/notifications",
			cfg.AuthorizeMiddleware(auth.NotificationsPublisher),
			cfg.Controller.CreateNotification)

		g.PATCH("/notifications/:id/status",
			cfg.AuthorizeMiddleware(auth.NotificationsPublisher),
			cfg.Controller.UpdateStatus)

		g.GET("/notifications/:id/status",
			cfg.AuthorizeMiddleware(auth.NotificationsPublisher, auth.Admin),
			cfg.Controller.GetStatus)

		g.POST("/notifications/:id/cancel",
			cfg.AuthorizeMiddleware(auth.NotificationsPublisher, auth.Admin),
			cfg.Controller.CancelDelivery)

		g.POST("/notifications/:id/recipients/statuses",
			cfg.AuthorizeMiddleware(auth.NotificationsPublisher),
			cfg.Controller.UpsertRecipientNotificationStatuses)

		g.GET("/notifications/:id/recipients/statuses",
			cfg.AuthorizeMiddleware(auth.NotificationsPublisher, auth.Admin),
			cfg.Controller.GetRecipientNotificationStatuses)

		g.DELETE("/notifications/:id",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.DeleteNotification)
	}

	return nil
}
