package routes

import (
	"github.com/gin-gonic/gin"

	c "github.com/notifique/controllers"
)

func SetupDistributionListRoutes(r *gin.Engine, dls c.DistributionListStorage) {

	controller := c.DistributionListController{Storage: dls}

	v0 := r.Group("/v0")
	{
		v0.GET("/distribution-lists", controller.GetDistributionLists)
		v0.POST("/distribution-lists", controller.CreateDistributionList)

		v0.PATCH("/distribution-lists/:id/recipients", controller.AddRecipients)
		v0.DELETE("/distribution-lists/:id/recipients", controller.DeleteRecipients)
		v0.GET("/distribution-lists/:id/recipients", controller.GetRecipients)
	}

}
