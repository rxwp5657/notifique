package routes

import (
	c "github.com/notifique/service/internal/controllers"
	"github.com/notifique/shared/auth"
)

type distributionListsRoutesCfg struct {
	routeGroupCfg
	Controller *c.DistributionListController
}

func SetupDistributionListRoutes(cfg distributionListsRoutesCfg) error {

	g := cfg.Engine.Group(cfg.Version, cfg.CacheMiddleware)
	{
		g.GET("/distribution-lists",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.GetDistributionLists)

		g.POST("/distribution-lists",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.CreateDistributionList)

		g.GET("/distribution-lists/:name/recipients",
			cfg.AuthorizeMiddleware(auth.Admin, auth.NotificationsPublisher),
			cfg.Controller.GetRecipients)

		g.PATCH("/distribution-lists/:name/recipients",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.AddRecipients)

		g.DELETE("/distribution-lists/:name/recipients",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.DeleteRecipients)

		g.DELETE("/distribution-lists/:name",
			cfg.AuthorizeMiddleware(auth.Admin),
			cfg.Controller.DeleteDistributionList)
	}

	return nil
}
