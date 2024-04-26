package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/go-playground/validator/v10"

	c "github.com/notifique/controllers"
	"github.com/notifique/internal"
)

func SetupDistributionListRoutes(r *gin.Engine, dls c.DistributionListStorage) {

	controller := c.DistributionListController{Storage: dls}

	v0 := r.Group("/v0")
	{
		v0.GET("/distribution-lists", controller.GetDistributionLists)
		v0.POST("/distribution-lists", controller.CreateDistributionList)
		v0.DELETE("/distribution-lists/:name", controller.DeleteDistributionList)

		v0.GET("/distribution-lists/:name/recipients", controller.GetRecipients)
		v0.PATCH("/distribution-lists/:name/recipients", controller.AddRecipients)
		v0.DELETE("/distribution-lists/:name/recipients", controller.DeleteRecipients)
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("distributionListName", internal.DLNameValidator)
	}
}
