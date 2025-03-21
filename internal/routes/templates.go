package routes

import (
	c "github.com/notifique/internal/controllers"
)

type templatesRoutesCfg struct {
	routeGroupCfg
	Controller *c.NotificationTemplateController
}

func SetupNotificationTemplateRoutes(cfg templatesRoutesCfg) error {

	g := cfg.Engine.Group(cfg.Version, cfg.Middlewares...)
	{
		g.POST("/notifications/templates", cfg.Controller.CreateNotificationTemplate)
		g.GET("/notifications/templates", cfg.Controller.GetTemplates)
		g.GET("/notifications/templates/:id", cfg.Controller.GetTemplateDetails)
		g.DELETE("/notifications/templates/:id", cfg.Controller.DeleteTemplate)
	}

	return nil
}
