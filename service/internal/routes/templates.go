package routes

import (
	c "github.com/notifique/service/internal/controllers"
	"github.com/notifique/shared/auth"
)

type templatesRoutesCfg struct {
	routeGroupCfg
	Controller *c.NotificationTemplateController
}

func SetupNotificationTemplateRoutes(cfg templatesRoutesCfg) error {

	g := cfg.Engine.Group(cfg.Version, cfg.CacheMiddleware)
	{
		g.POST("/notifications/templates",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.CreateNotificationTemplate)

		g.GET("/notifications/templates",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.GetTemplates)

		g.GET("/notifications/templates/:id",
			cfg.AuthorizeMiddleware(auth.Admin, auth.NotificationsPublisher),
			cfg.Controller.GetTemplateDetails)

		g.DELETE("/notifications/templates/:id",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.DeleteTemplate)
	}

	return nil
}
