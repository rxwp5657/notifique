package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/go-playground/validator/v10"

	"github.com/notifique/internal/server"
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

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("distributionListName", server.DLNameValidator)
	}
}
