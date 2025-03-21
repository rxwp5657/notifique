package routes

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/go-redis/redis_rate/v10"
	redis "github.com/redis/go-redis/v9"

	"github.com/notifique/internal"
	"github.com/notifique/internal/cache"
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
	GetCacheTTL() (*int, error)
}

type EngineConfig struct {
	RedisClient        *redis.Client
	Registry           Registry
	Cache              cache.Cache
	Publisher          controllers.NotificationPublisher
	Broker             controllers.UserNotificationBroker
	EngineConfigurator EngineConfigurator
}

type routeGroupCfg struct {
	Engine      *gin.Engine
	Version     string
	Middlewares []gin.HandlerFunc
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
		Cache:    cfg.Cache,
	}

	uc := controllers.UserController{
		Registry: cfg.Registry,
		Broker:   cfg.Broker,
		Cache:    cfg.Cache,
	}

	ntc := controllers.NotificationTemplateController{
		Registry: cfg.Registry,
		Cache:    cfg.Cache,
	}

	r := gin.Default()

	r.Use(gin.Recovery())

	expectedHost := cfg.EngineConfigurator.GetExpectedHost()
	r.Use(middleware.Security(expectedHost))

	ttl, err := cfg.EngineConfigurator.GetCacheTTL()

	if err != nil {
		return nil, err
	}

	var cacheMiddlware gin.HandlerFunc

	if cfg.Cache != nil && ttl != nil {
		cacheMiddlware = middleware.GetCache(cfg.Cache, time.Duration(*ttl)*time.Second)
	}

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

	middlewares := []gin.HandlerFunc{}

	if cacheMiddlware != nil {
		middlewares = append(middlewares, cacheMiddlware)
	}

	routesCfg := routeGroupCfg{
		Engine:      r,
		Version:     version,
		Middlewares: middlewares,
	}

	_ = SetupNotificationRoutes(r, version, &nc)

	_ = SetupDistributionListRoutes(distributionListsRoutesCfg{
		routeGroupCfg: routesCfg,
		Controller:    &dlc,
	})

	_ = SetupUsersRoutes(usersRoutesCfg{
		routeGroupCfg: routesCfg,
		Controller:    &uc,
	})

	_ = SetupNotificationTemplateRoutes(templatesRoutesCfg{
		routeGroupCfg: routesCfg,
		Controller:    &ntc,
	})

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("distributionlistname", internal.DLNameValidator)
		v.RegisterValidation("unique_var_name", internal.UniqueTemplateVarValidator)
		v.RegisterValidation("future", internal.FutureValidator)
		v.RegisterValidation("templatevarname", internal.TemplateNameValidator)
	}

	return r, nil
}
