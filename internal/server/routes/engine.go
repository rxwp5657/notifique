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

type Cache interface {
	controllers.NotificationCache
}

type EngineConfig struct {
	Registry            Registry
	Cache               Cache
	Publisher           controllers.NotificationPublisher
	Broker              controllers.UserNotificationBroker
	VersionConfigurator VersionConfigurator
}

func NewEngine(cfg EngineConfig) (*gin.Engine, error) {

	version, err := cfg.VersionConfigurator.GetVersion()

	if err != nil {
		return nil, err
	}

	match, _ := regexp.MatchString(versionRegex, version)

	if !match {
		return nil, fmt.Errorf("api version should have the format %s", versionRegex)
	}

	nc := controllers.NotificationController{
		Registry:  cfg.Registry,
		Publisher: cfg.Publisher,
		Cache:     cfg.Cache,
	}

	dlc := controllers.DistributionListController{
		Registry: cfg.Registry,
	}

	uc := controllers.UserController{
		Registry: cfg.Registry,
		Broker:   cfg.Broker,
	}

	ntc := controllers.NotificationTemplateController{
		Registry: cfg.Registry,
	}

	r := gin.Default()

	_ = SetupNotificationRoutes(r, version, &nc)
	_ = SetupDistributionListRoutes(r, version, &dlc)
	_ = SetupUsersRoutes(r, version, &uc)
	_ = SetupNotificationTemplateRoutes(r, version, &ntc)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("distributionListName", server.DLNameValidator)
		v.RegisterValidation("unique_var_name", server.UniqueTemplateVarValidator)
		v.RegisterValidation("future", server.FutureValidator)
	}

	return r, nil
}
