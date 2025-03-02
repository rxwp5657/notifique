package routes

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/controllers"
)

const versionRegex = "(/v[0-9]{1,2}|^$)"

type Registry interface {
	controllers.NotificationRegistry
	controllers.UserRegistry
	controllers.DistributionRegistry
	controllers.NotificationTemplateRegistry
}

type VersionConfigurator interface {
	GetVersion() (string, error)
}

func NewEngine(registry Registry, pub controllers.NotificationPublisher, bk controllers.UserNotificationBroker, vc VersionConfigurator) (*gin.Engine, error) {

	version, err := vc.GetVersion()

	if err != nil {
		return nil, err
	}

	match, _ := regexp.MatchString(versionRegex, version)

	if !match {
		return nil, fmt.Errorf("api version should have the format %s", versionRegex)
	}

	r := gin.Default()

	SetupNotificationRoutes(r, version, registry, pub)
	SetupDistributionListRoutes(r, version, registry)
	SetupUsersRoutes(r, version, registry, bk)
	SetupNotificationTemplateRoutes(r, version, registry)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("distributionListName", server.DLNameValidator)
		v.RegisterValidation("unique_var_name", server.UniqueTemplateVarValidator)
		v.RegisterValidation("future", server.FutureValidator)
	}

	return r, nil
}
