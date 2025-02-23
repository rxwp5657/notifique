package routes

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
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

	return r, nil
}
