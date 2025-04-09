//go:build wireinject
// +build wireinject

package di

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/google/wire"
	"github.com/notifique/service/internal/controllers"
	"github.com/notifique/service/internal/middleware"
	"github.com/notifique/service/internal/routes"
	"github.com/notifique/service/pkg/deployments"
	"github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"

	bk "github.com/notifique/service/internal/broker"
	cfg "github.com/notifique/service/internal/config"
	pub "github.com/notifique/service/internal/publish"
	dynamoregistry "github.com/notifique/service/internal/registry/dynamodb"
	pg "github.com/notifique/service/internal/registry/postgres"
	tcfg "github.com/notifique/service/internal/testutils/config"
	c "github.com/notifique/service/internal/testutils/containers"
	mk "github.com/notifique/service/internal/testutils/mocks"
	cache "github.com/notifique/shared/cache"
	"github.com/notifique/shared/clients"
	sc "github.com/notifique/shared/containers"
)

type PostgresMockedPubIntegrationTest struct {
	Postgres  *sc.Postgres
	Redis     *sc.Redis
	Registry  *pg.Registry
	Publisher *mk.MockNotificationPublisher
	Broker    *bk.Redis
	Engine    *gin.Engine
}

type DynamoMockedPubIntegrationTest struct {
	Dynamo    *sc.Dynamo
	Redis     *sc.Redis
	Registry  *dynamoregistry.Registry
	Publisher *mk.MockNotificationPublisher
	Broker    *bk.Redis
	Engine    *gin.Engine
}

type PgSQSPriorityIntegrationTest struct {
	Postgres  *sc.Postgres
	SQS       *c.SQSPriority
	Redis     *sc.Redis
	Registry  *pg.Registry
	Publisher *pub.Priority
	Cache     *cache.Redis
}

type PgRabbitMQPriorityIntegrationTest struct {
	Postgres  *sc.Postgres
	RabbitMQ  *c.RabbitMQPriority
	Redis     *sc.Redis
	Registry  *pg.Registry
	Publisher *pub.Priority
	Cache     *cache.Redis
}

type DynamoSQSPriorityIntegrationTest struct {
	Dynamo    *sc.Dynamo
	SQS       *c.SQSPriority
	Registry  *dynamoregistry.Registry
	Publisher *pub.Priority
}

type DynamoRabbitMQPriorityIntegrationTest struct {
	Dynamo    *sc.Dynamo
	RabbitMQ  *c.RabbitMQPriority
	Registry  *dynamoregistry.Registry
	Publisher *pub.Priority
}

type MockedBackend struct {
	Registry  *mk.MockedRegistry
	Publisher *mk.MockNotificationPublisher
	Broker    *mk.MockUserNotificationBroker
	Cache     *mk.MockCache
	Engine    *gin.Engine
}

var DynamoSet = wire.NewSet(
	clients.NewDynamoDBClient,
	dynamoregistry.NewDynamoDBRegistry,
	wire.Bind(new(dynamoregistry.DynamoDBAPI), new(*dynamodb.Client)),
	wire.Bind(new(routes.Registry), new(*dynamoregistry.Registry)),
	wire.Bind(new(controllers.NotificationRegistry), new(*dynamoregistry.Registry)),
)

var PostgresSet = wire.NewSet(
	pg.NewPostgresRegistry,
	wire.Bind(new(routes.Registry), new(*pg.Registry)),
	wire.Bind(new(controllers.NotificationRegistry), new(*pg.Registry)),
)

var SQSPublisherSet = wire.NewSet(
	clients.NewSQSClient,
	pub.NewSQSPublisher,
	wire.Bind(new(pub.SQSAPI), new(*sqs.Client)),
	wire.Bind(new(pub.Publisher), new(*pub.SQS)),
)

var RabbitMQPublisherSet = wire.NewSet(
	clients.NewRabbitMQClient,
	pub.NewRabbitMQPublisher,
	wire.Bind(new(pub.RabbitMQAPI), new(*clients.RabbitMQ)),
	wire.Bind(new(pub.Publisher), new(*pub.RabbitMQ)),
)

var PriorityPublisherCfgSet = wire.NewSet(
	wire.Struct(new(pub.PriorityPublisherCfg), "*"),
)

var RedisSet = wire.NewSet(
	cache.NewRedisClient,
	wire.Bind(new(cache.CacheRedisApi), new(*redis.Client)),
	wire.Bind(new(bk.BrokerRedisApi), new(*redis.Client)),
)

var RedisCacheSet = wire.NewSet(
	cache.NewRedisCache,
	wire.Bind(new(cache.Cache), new(*cache.Redis)),
)

var RedisUserNotificationBrokerSet = wire.NewSet(
	bk.NewRedisBroker,
	wire.Bind(new(controllers.UserNotificationBroker), new(*bk.Redis)),
)

var PrioritySet = wire.NewSet(
	PriorityPublisherCfgSet,
	pub.NewPriorityPublisher,
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.Priority)),
)

var PostgresContainerSet = wire.NewSet(
	sc.NewPostgresContainer,
	wire.Bind(new(clients.PostgresConfigurator), new(*sc.Postgres)),
)

var SQSPriorityContainerSet = wire.NewSet(
	c.NewSQSPriorityContainer,
	wire.Bind(new(clients.SQSConfigurator), new(*c.SQSPriority)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*c.SQSPriority)),
)

var RabbitMQPriorityContainerSet = wire.NewSet(
	c.NewRabbitMQPriorityContainer,
	wire.Bind(new(clients.RabbitMQConfigurator), new(*c.RabbitMQPriority)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*c.RabbitMQPriority)),
)

var DynamoContainerSet = wire.NewSet(
	sc.NewDynamoContainer,
	wire.Bind(new(clients.DynamoConfigurator), new(*sc.Dynamo)),
)

var RedisContainerSet = wire.NewSet(
	sc.NewRedisContainer,
	wire.Bind(new(cache.RedisConfigurator), new(*sc.Redis)),
	wire.Bind(new(bk.BrokerConfigurator), new(*sc.Redis)),
)

var RedisRateSet = wire.NewSet(
	middleware.NewRedisLimiter,
	wire.Bind(new(middleware.RateLimiter), new(*redis_rate.Limiter)),
)

var MiddlewareSet = wire.NewSet(
	RedisRateSet,
	wire.Struct(new(middleware.RateLimitCfg), "*"),
	wire.Struct(new(middleware.CacheCfg), "*"),
	middleware.NewSecurityMiddleware,
	middleware.NewRateLimitMiddleware,
	middleware.NewCacheMiddleware,
	middleware.NewAuthMiddleware,
	wire.Value(middleware.Authorize),
)

var MockedPublihserSet = wire.NewSet(
	mk.NewMockNotificationPublisher,
	wire.Bind(new(controllers.NotificationPublisher), new(*mk.MockNotificationPublisher)),
)

var MockedDistributionRegistrySet = wire.NewSet(
	mk.NewMockDistributionRegistry,
	wire.Bind(new(controllers.DistributionRegistry), new(*mk.MockDistributionRegistry)),
)

var MockedUserRegistrySet = wire.NewSet(
	mk.NewMockUserRegistry,
	wire.Bind(new(controllers.UserRegistry), new(*mk.MockUserRegistry)),
)

var MockedNotificationRegistrySet = wire.NewSet(
	mk.NewMockNotificationRegistry,
	wire.Bind(new(controllers.NotificationRegistry), new(*mk.MockNotificationRegistry)),
)

var MockedNotificationTemplateRegistrySet = wire.NewSet(
	mk.NewMockNotificationTemplateRegistry,
	wire.Bind(new(controllers.NotificationTemplateRegistry), new(*mk.MockNotificationTemplateRegistry)),
)

var MockedUserNotificationBroker = wire.NewSet(
	mk.NewMockUserNotificationBroker,
	wire.Bind(new(controllers.UserNotificationBroker), new(*mk.MockUserNotificationBroker)),
)

var MockedRegistrySet = wire.NewSet(
	MockedDistributionRegistrySet,
	MockedUserRegistrySet,
	MockedNotificationRegistrySet,
	MockedNotificationTemplateRegistrySet,
	mk.NewMockedRegistry,
	wire.Bind(new(routes.Registry), new(*mk.MockedRegistry)),
)

var MockedMiddlewareSet = wire.NewSet(
	mk.NewTestAuthMiddleware,
	mk.NewTestCacheMiddleware,
	mk.NewTestSecurityMiddleware,
	mk.NewTestRateLimitMiddleware,
	wire.Value(middleware.Authorize),
)

var TestVersionConfiguratorSet = wire.NewSet(
	tcfg.NewTestVersionConfigurator,
	wire.Bind(new(routes.EngineConfigurator), new(tcfg.TestEngineConfigurator)),
)

var EnvConfigSet = wire.NewSet(
	cfg.NewEnvConfig,
	wire.Bind(new(clients.PostgresConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(clients.DynamoConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(clients.SQSConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(clients.RabbitMQConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.RabbitMQPriorityConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.SQSPriorityConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(cache.RedisConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(bk.BrokerConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(routes.EngineConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(middleware.AuthConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(middleware.CacheConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(middleware.RateLimitConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(middleware.SecurityConfigurator), new(*cfg.EnvConfig)),
)

var MockedCacheSet = wire.NewSet(
	mk.NewMockCache,
	wire.Bind(new(cache.Cache), new(*mk.MockCache)),
)

var EngineConfigSet = wire.NewSet(
	wire.Struct(new(routes.EngineConfig), "*"),
)

func InjectPgPrioritySQS(envfile *string) (*gin.Engine, error) {

	wire.Build(
		EnvConfigSet,
		PostgresSet,
		SQSPublisherSet,
		RedisSet,
		RedisCacheSet,
		PrioritySet,
		RedisUserNotificationBrokerSet,
		MiddlewareSet,
		EngineConfigSet,
		routes.NewEngine,
	)

	return nil, nil
}

func InjectPgPriorityRabbitMQ(envfile *string) (*gin.Engine, func(), error) {

	wire.Build(
		EnvConfigSet,
		PostgresSet,
		RabbitMQPublisherSet,
		RedisSet,
		RedisCacheSet,
		PrioritySet,
		RedisUserNotificationBrokerSet,
		MiddlewareSet,
		EngineConfigSet,
		routes.NewEngine,
	)

	return nil, nil, nil
}

func InjectDynamoPrioritySQS(envfile *string) (*gin.Engine, error) {

	wire.Build(
		EnvConfigSet,
		DynamoSet,
		SQSPublisherSet,
		RedisSet,
		RedisCacheSet,
		PrioritySet,
		RedisUserNotificationBrokerSet,
		MiddlewareSet,
		EngineConfigSet,
		routes.NewEngine,
	)

	return nil, nil
}

func InjectDynamoPriorityRabbitMQ(envfile *string) (*gin.Engine, func(), error) {

	wire.Build(
		EnvConfigSet,
		DynamoSet,
		RabbitMQPublisherSet,
		RedisSet,
		RedisCacheSet,
		PrioritySet,
		RedisUserNotificationBrokerSet,
		MiddlewareSet,
		EngineConfigSet,
		routes.NewEngine,
	)

	return nil, nil, nil
}

func InjectPgSQSPriorityIntegrationTest(ctx context.Context) (*PgSQSPriorityIntegrationTest, func(), error) {

	wire.Build(
		PostgresContainerSet,
		SQSPriorityContainerSet,
		RedisContainerSet,
		PostgresSet,
		RedisSet,
		RedisCacheSet,
		PrioritySet,
		SQSPublisherSet,
		wire.Struct(new(PgSQSPriorityIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectPgRabbitMQPriorityIntegrationTest(ctx context.Context) (*PgRabbitMQPriorityIntegrationTest, func(), error) {

	wire.Build(
		PostgresContainerSet,
		RabbitMQPriorityContainerSet,
		RedisContainerSet,
		PostgresSet,
		RedisSet,
		RedisCacheSet,
		PrioritySet,
		RabbitMQPublisherSet,
		wire.Struct(new(PgRabbitMQPriorityIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectMockedBackend(ctx context.Context, mockController *gomock.Controller) (*MockedBackend, error) {

	wire.Build(
		TestVersionConfiguratorSet,
		MockedRegistrySet,
		MockedPublihserSet,
		MockedUserNotificationBroker,
		MockedCacheSet,
		MockedMiddlewareSet,
		EngineConfigSet,
		wire.Value((*redis.Client)(nil)),
		routes.NewEngine,
		wire.Struct(new(MockedBackend), "*"),
	)

	return nil, nil
}

func InjectRabbitMQPriorityDeployer(envfile *string) (*deployments.RabbitMQPriorityDeployer, func(), error) {

	wire.Build(
		EnvConfigSet,
		deployments.NewRabbitMQPriorityDeployer,
	)

	return nil, nil, nil
}

func InjectSQSPriorityDeployer(envfile *string) (*deployments.SQSPriorityDeployer, func(), error) {

	wire.Build(
		EnvConfigSet,
		deployments.NewSQSPriorityDeployer,
	)

	return nil, nil, nil
}
