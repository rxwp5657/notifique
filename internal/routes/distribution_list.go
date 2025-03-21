package routes

import (
	c "github.com/notifique/internal/controllers"
)

type distributionListsRoutesCfg struct {
	routeGroupCfg
	Controller *c.DistributionListController
}

func SetupDistributionListRoutes(cfg distributionListsRoutesCfg) error {

	g := cfg.Engine.Group(cfg.Version, cfg.Middlewares...)
	{
		g.GET("/distribution-lists", cfg.Controller.GetDistributionLists)
		g.POST("/distribution-lists", cfg.Controller.CreateDistributionList)

		g.GET("/distribution-lists/:name/recipients", cfg.Controller.GetRecipients)
		g.PATCH("/distribution-lists/:name/recipients", cfg.Controller.AddRecipients)
		g.DELETE("/distribution-lists/:name/recipients", cfg.Controller.DeleteRecipients)
		g.DELETE("/distribution-lists/:name", cfg.Controller.DeleteDistributionList)
	}

	return nil
}
