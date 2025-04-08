package routes

import (
	c "github.com/notifique/service/internal/controllers"
	"github.com/notifique/shared/auth"
)

type usersRoutesCfg struct {
	routeGroupCfg
	Controller *c.UserController
}

func SetupUsersRoutes(cfg usersRoutesCfg) error {

	g := cfg.Engine.Group(cfg.Version, cfg.CacheMiddleware)
	{
		g.GET("/users/me/notifications",
			cfg.AuthorizeMiddleware(auth.User),
			cfg.Controller.GetUserNotifications)

		g.GET("/users/me/notifications/live",
			cfg.AuthorizeMiddleware(auth.User),
			cfg.Controller.GetLiveUserNotifications)

		g.PATCH("/users/me/notifications/:id",
			cfg.AuthorizeMiddleware(auth.User),
			cfg.Controller.SetReadStatus)

		g.GET("/users/me/notifications/config",
			cfg.AuthorizeMiddleware(auth.User),
			cfg.Controller.GetUserConfig)

		g.PUT("/users/me/notifications/config",
			cfg.AuthorizeMiddleware(auth.User),
			cfg.Controller.UpdateUserConfig)

		g.POST("/users/notifications",
			cfg.AuthorizeMiddleware(auth.UserNotificationPublisher),
			cfg.Controller.CreateNotifications)
	}

	return nil
}
