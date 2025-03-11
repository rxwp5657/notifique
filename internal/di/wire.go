//go:build wireinject
// +build wireinject

package di

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/notifique/internal/deployments"
	"github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/routes"
	"github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"

	bk "github.com/notifique/internal/broker"
	cache "github.com/notifique/internal/cache"
	cfg "github.com/notifique/internal/config"
	pub "github.com/notifique/internal/publish"
	dynamoregistry "github.com/notifique/internal/registry/dynamodb"
	pg "github.com/notifique/internal/registry/postgres"
	tcfg "github.com/notifique/internal/testutils/config"
	c "github.com/notifique/internal/testutils/containers"
	mk "github.com/notifique/internal/testutils/mocks"
)

type PostgresMockedPubIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	RedisContainer    *c.RedisContainer
	Registry          *pg.Registry
	Publisher         *mk.MockNotificationPublisher
	Broker            *bk.Redis
	Engine            *gin.Engine
}

type DynamoMockedPubIntegrationTest struct {
	DynamoContainer *c.DynamoContainer
	RedisContainer  *c.RedisContainer
	Registry        *dynamoregistry.Registry
	Publisher       *mk.MockNotificationPublisher
	Broker          *bk.Redis
	Engine          *gin.Engine
}

type PgSQSPriorityIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	SQSContainer      *c.SQSPriorityContainer
	RedisContainer    *c.RedisContainer
	Registry          *pg.Registry
	Publisher         *pub.PriorityPublisher
	Cache             *cache.RedisCache
}

type PgRabbitMQPriorityIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	RedisContainer    *c.RedisContainer
	Registry          *pg.Registry
	Publisher         *pub.PriorityPublisher
	Cache             *cache.RedisCache
}

type DynamoSQSPriorityIntegrationTest struct {
	DynamoContainer *c.DynamoContainer
	SQSContainer    *c.SQSPriorityContainer
	Registry        *dynamoregistry.Registry
	Publisher       *pub.PriorityPublisher
}

type DynamoRabbitMQPriorityIntegrationTest struct {
	DynamoContainer   *c.DynamoContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	Registry          *dynamoregistry.Registry
	Publisher         *pub.PriorityPublisher
}

type MockedBackend struct {
	Registry  *mk.MockedRegistry
	Publisher *mk.MockNotificationPublisher
	Broker    *mk.MockUserNotificationBroker
	Cache     *mk.MockNotificationCache
	Engine    *gin.Engine
}

var DynamoSet = wire.NewSet(
	dynamoregistry.NewDynamoDBClient,
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
	pub.NewSQSClient,
	pub.NewSQSPublisher,
	wire.Bind(new(pub.SQSAPI), new(*sqs.Client)),
	wire.Bind(new(pub.Publisher), new(*pub.SQSPublisher)),
)

var RabbitMQPublisherSet = wire.NewSet(
	pub.NewRabbitMQClient,
	pub.NewRabbitMQPublisher,
	wire.Bind(new(pub.RabbitMQAPI), new(*pub.RabbitMQClient)),
	wire.Bind(new(pub.Publisher), new(*pub.RabbitMQPublisher)),
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
	wire.Bind(new(controllers.NotificationCache), new(*cache.RedisCache)),
	wire.Bind(new(routes.Cache), new(*cache.RedisCache)),
)

var RedisUserNotificationBrokerSet = wire.NewSet(
	bk.NewRedisBroker,
	wire.Bind(new(controllers.UserNotificationBroker), new(*bk.Redis)),
)

var PrioritySet = wire.NewSet(
	PriorityPublisherCfgSet,
	pub.NewPriorityPublisher,
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.PriorityPublisher)),
)

var PostgresContainerSet = wire.NewSet(
	c.NewPostgresContainer,
	wire.Bind(new(pg.PostgresConfigurator), new(*c.PostgresContainer)),
)

var SQSPriorityContainerSet = wire.NewSet(
	c.NewSQSPriorityContainer,
	wire.Bind(new(pub.SQSConfigurator), new(*c.SQSPriorityContainer)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*c.SQSPriorityContainer)),
)

var RabbitMQPriorityContainerSet = wire.NewSet(
	c.NewRabbitMQPriorityContainer,
	wire.Bind(new(pub.RabbitMQConfigurator), new(*c.RabbitMQPriorityContainer)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*c.RabbitMQPriorityContainer)),
)

var DynamoContainerSet = wire.NewSet(
	c.NewDynamoContainer,
	wire.Bind(new(dynamoregistry.DynamoConfigurator), new(*c.DynamoContainer)),
)

var RedisContainerSet = wire.NewSet(
	c.NewRedisContainer,
	wire.Bind(new(cache.RedisConfigurator), new(*c.RedisContainer)),
	wire.Bind(new(bk.BrokerConfigurator), new(*c.RedisContainer)),
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

var TestVersionConfiguratorSet = wire.NewSet(
	tcfg.NewTestVersionConfigurator,
	wire.Bind(new(routes.EngineConfigurator), new(tcfg.TestEngineConfigurator)),
)

var EnvConfigSet = wire.NewSet(
	cfg.NewEnvConfig,
	wire.Bind(new(pg.PostgresConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(dynamoregistry.DynamoConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.SQSConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.RabbitMQConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.RabbitMQPriorityConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.SQSPriorityConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(cache.RedisConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(bk.BrokerConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(routes.EngineConfigurator), new(*cfg.EnvConfig)),
)

var MockedNotificationCacheSet = wire.NewSet(
	mk.NewMockNotificationCache,
	wire.Bind(new(routes.Cache), new(*mk.MockNotificationCache)),
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
		MockedNotificationCacheSet,
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
