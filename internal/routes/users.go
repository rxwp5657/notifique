package routes

import (
	c "github.com/notifique/internal/controllers"
)

type usersRoutesCfg struct {
	routeGroupCfg
	Controller *c.UserController
}

func SetupUsersRoutes(cfg usersRoutesCfg) error {

	g := cfg.Engine.Group(cfg.Version, cfg.Middlewares...)
	{
		g.GET("/users/me/notifications", cfg.Controller.GetUserNotifications)
		g.GET("/users/me/notifications/live", cfg.Controller.GetLiveUserNotifications)
		g.PATCH("/users/me/notifications/:id", cfg.Controller.SetReadStatus)

		g.GET("/users/me/notifications/config", cfg.Controller.GetUserConfig)
		g.PUT("/users/me/notifications/config", cfg.Controller.UpdateUserConfig)
	}

	return nil
}
