package routes

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/go-redis/redis_rate/v10"
	redis "github.com/redis/go-redis/v9"

	"github.com/notifique/internal"
	"github.com/notifique/internal/controllers"
	"github.com/notifique/internal/middleware"
)

const versionRegex = "(/v[0-9]{1,2}|^$)"

type Registry interface {
	controllers.NotificationRegistry
	controllers.UserRegistry
	controllers.DistributionRegistry
	controllers.NotificationTemplateRegistry
}

type EngineConfigurator interface {
	GetVersion() (string, error)
	GetExpectedHost() *string
	GetRequestsPerSecond() (*int, error)
}

type Cache interface {
	controllers.NotificationCache
}

type EngineConfig struct {
	RedisClient        *redis.Client
	Registry           Registry
	Cache              Cache
	Publisher          controllers.NotificationPublisher
	Broker             controllers.UserNotificationBroker
	EngineConfigurator EngineConfigurator
}

func NewEngine(cfg EngineConfig) (*gin.Engine, error) {

	version, err := cfg.EngineConfigurator.GetVersion()

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

	expectedHost := cfg.EngineConfigurator.GetExpectedHost()
	r.Use(middleware.Security(expectedHost))

	requestsPerSecond, err := cfg.EngineConfigurator.GetRequestsPerSecond()

	if err != nil {
		return nil, err
	}

	if cfg.RedisClient != nil && requestsPerSecond != nil {
		if *requestsPerSecond <= 0 {
			return nil, fmt.Errorf("requests per second should be greater than 0")
		}

		rateLimitier := redis_rate.NewLimiter(cfg.RedisClient)
		r.Use(middleware.RateLimit(rateLimitier, *requestsPerSecond))
	}

	_ = SetupNotificationRoutes(r, version, &nc)
	_ = SetupDistributionListRoutes(r, version, &dlc)
	_ = SetupUsersRoutes(r, version, &uc)
	_ = SetupNotificationTemplateRoutes(r, version, &ntc)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("distributionlistname", internal.DLNameValidator)
		v.RegisterValidation("unique_var_name", internal.UniqueTemplateVarValidator)
		v.RegisterValidation("future", internal.FutureValidator)
		v.RegisterValidation("templatevarname", internal.TemplateNameValidator)
	}

	return r, nil
}
