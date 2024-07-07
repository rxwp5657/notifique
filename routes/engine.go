package routes

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/notifique/controllers"
)

const versionRegex = "(/v[0-9]{1,2}|^$)"

type Storage interface {
	controllers.NotificationStorage
	controllers.UserStorage
	controllers.DistributionListStorage
}

type VersionConfigurator interface {
	GetVersion() (string, error)
}

func NewEngine(storage Storage, pub controllers.NotificationPublisher, bk controllers.UserNotificationBroker, vc VersionConfigurator) (*gin.Engine, error) {

	version, err := vc.GetVersion()

	if err != nil {
		return nil, err
	}

	match, _ := regexp.MatchString(versionRegex, version)

	if !match {
		return nil, fmt.Errorf("api version should have the format %s", versionRegex)
	}

	r := gin.Default()

	SetupNotificationRoutes(r, version, storage, pub)
	SetupDistributionListRoutes(r, version, storage)
	SetupUsersRoutes(r, version, storage, bk)

	return r, nil
}
