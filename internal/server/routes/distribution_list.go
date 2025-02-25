package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/internal/server/controllers"
)

func SetupDistributionListRoutes(r *gin.Engine, version string, dls c.DistributionRegistry) {

	controller := c.DistributionListController{Registry: dls}

	g := r.Group(version)
	{
		g.GET("/distribution-lists", controller.GetDistributionLists)
		g.POST("/distribution-lists", controller.CreateDistributionList)
		g.DELETE("/distribution-lists/:name", controller.DeleteDistributionList)

		g.GET("/distribution-lists/:name/recipients", controller.GetRecipients)
		g.PATCH("/distribution-lists/:name/recipients", controller.AddRecipients)
		g.DELETE("/distribution-lists/:name/recipients", controller.DeleteRecipients)
	}
}
