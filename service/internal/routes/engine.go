package routes

import (
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	redis "github.com/redis/go-redis/v9"

	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/controllers"
	"github.com/notifique/service/internal/middleware"
	"github.com/notifique/shared/auth"
	"github.com/notifique/shared/cache"
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
}

type EngineConfig struct {
	RedisClient        *redis.Client
	Registry           Registry
	Cache              cache.Cache
	Publisher          controllers.NotificationPublisher
	Broker             controllers.UserNotificationBroker
	EngineConfigurator EngineConfigurator
	Authorize          func(...auth.Scope) gin.HandlerFunc
	Authenticate       middleware.AuthMiddleware
	RateLimit          middleware.RateLimitMiddleware
	CacheMiddleware    middleware.CacheMiddleware
	SecurityMiddleware middleware.SecurityMiddleware
}

type routeGroupCfg struct {
	Engine              *gin.Engine
	Version             string
	CacheMiddleware     gin.HandlerFunc
	AuthorizeMiddleware func(...auth.Scope) gin.HandlerFunc
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

	nc = controllers.NotificationController{
		Registry:  cfg.Registry,
		Publisher: cfg.Publisher,
		Cache:     cfg.Cache,
	}

	r := gin.Default()

	r.Use(gin.Recovery())
	r.Use(gin.HandlerFunc(cfg.SecurityMiddleware))
	r.Use(gin.HandlerFunc(cfg.RateLimit))
	r.Use(gin.HandlerFunc(cfg.Authenticate))
	r.Use(gin.HandlerFunc(cfg.RateLimit))

	routesCfg := routeGroupCfg{
		Engine:              r,
		Version:             version,
		CacheMiddleware:     gin.HandlerFunc(cfg.CacheMiddleware),
		AuthorizeMiddleware: cfg.Authorize,
	}

	_ = SetupNotificationRoutes(notificationsRoutesCfg{
		routeGroupCfg: routesCfg,
		Controller:    &nc,
	})

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
