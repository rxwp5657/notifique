//go:build wireinject
// +build wireinject

package di

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/routes"
	"github.com/redis/go-redis/v9"
	gomock "go.uber.org/mock/gomock"

	bk "github.com/notifique/internal/broker"
	cache "github.com/notifique/internal/cache"
	cfg "github.com/notifique/internal/config"
	"github.com/notifique/internal/deployments"
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
	Registry          *pg.Registry
	Publisher         *pub.PriorityPublisher
}

type PgRabbitMQPriorityIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	Registry          *pg.Registry
	Publisher         *pub.PriorityPublisher
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

var PostgresSQSPriroritySet = wire.NewSet(
	PostgresSet,
	SQSPublisherSet,
	pub.NewPriorityPublisher,
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.PriorityPublisher)),
)

var PostgresRabbitMQPriroritySet = wire.NewSet(
	PostgresSet,
	RabbitMQPublisherSet,
	pub.NewPriorityPublisher,
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.PriorityPublisher)),
)

var DynamoSQSPriroritySet = wire.NewSet(
	DynamoSet,
	SQSPublisherSet,
	pub.NewPriorityPublisher,
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.PriorityPublisher)),
)

var DynamoRabbitMQPriroritySet = wire.NewSet(
	DynamoSet,
	RabbitMQPublisherSet,
	pub.NewPriorityPublisher,
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.PriorityPublisher)),
)

var RedisUserNotificationBrokerSet = wire.NewSet(
	cache.NewRedisClient,
	bk.NewRedisBroker,
	wire.Bind(new(bk.RedisApi), new(*redis.Client)),
	wire.Bind(new(controllers.UserNotificationBroker), new(*bk.Redis)),
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
	wire.Bind(new(routes.VersionConfigurator), new(tcfg.TestVersionConfiguratorFunc)),
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
	wire.Bind(new(routes.VersionConfigurator), new(*cfg.EnvConfig)),
)

func InjectPgPrioritySQS(envfile string) (*gin.Engine, error) {

	wire.Build(
		EnvConfigSet,
		PostgresSQSPriroritySet,
		RedisUserNotificationBrokerSet,
		routes.NewEngine,
	)

	return nil, nil
}

func InjectPgPriorityRabbitMQ(envfile string) (*gin.Engine, func(), error) {

	wire.Build(
		EnvConfigSet,
		PostgresRabbitMQPriroritySet,
		RedisUserNotificationBrokerSet,
		routes.NewEngine,
	)

	return nil, nil, nil
}

func InjectDynamoPrioritySQS(envfile string) (*gin.Engine, error) {

	wire.Build(
		EnvConfigSet,
		DynamoSQSPriroritySet,
		RedisUserNotificationBrokerSet,
		routes.NewEngine,
	)

	return nil, nil
}

func InjectDynamoPriorityRabbitMQ(envfile string) (*gin.Engine, func(), error) {

	wire.Build(
		EnvConfigSet,
		DynamoRabbitMQPriroritySet,
		RedisUserNotificationBrokerSet,
		routes.NewEngine,
	)

	return nil, nil, nil
}

func InjectPgMockedPubIntegrationTest(ctx context.Context, mockController *gomock.Controller) (*PostgresMockedPubIntegrationTest, func(), error) {

	wire.Build(
		TestVersionConfiguratorSet,
		PostgresContainerSet,
		RedisContainerSet,
		PostgresSet,
		MockedPublihserSet,
		RedisUserNotificationBrokerSet,
		routes.NewEngine,
		wire.Struct(new(PostgresMockedPubIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectDynamoMockedPubIntegrationTest(ctx context.Context, mockController *gomock.Controller) (*DynamoMockedPubIntegrationTest, func(), error) {

	wire.Build(
		TestVersionConfiguratorSet,
		DynamoContainerSet,
		RedisContainerSet,
		DynamoSet,
		MockedPublihserSet,
		RedisUserNotificationBrokerSet,
		routes.NewEngine,
		wire.Struct(new(DynamoMockedPubIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectPgSQSPriorityIntegrationTest(ctx context.Context) (*PgSQSPriorityIntegrationTest, func(), error) {

	wire.Build(
		PostgresContainerSet,
		SQSPriorityContainerSet,
		PostgresSQSPriroritySet,
		wire.Struct(new(PgSQSPriorityIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectPgRabbitMQPriorityIntegrationTest(ctx context.Context) (*PgRabbitMQPriorityIntegrationTest, func(), error) {

	wire.Build(
		PostgresContainerSet,
		RabbitMQPriorityContainerSet,
		PostgresRabbitMQPriroritySet,
		wire.Struct(new(PgRabbitMQPriorityIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectDynamoSQSPriorityIntegrationTest(ctx context.Context) (*DynamoSQSPriorityIntegrationTest, func(), error) {

	wire.Build(
		DynamoContainerSet,
		SQSPriorityContainerSet,
		DynamoSQSPriroritySet,
		wire.Struct(new(DynamoSQSPriorityIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectDynamoRabbitMQPriorityIntegrationTest(ctx context.Context) (*DynamoRabbitMQPriorityIntegrationTest, func(), error) {

	wire.Build(
		DynamoContainerSet,
		RabbitMQPriorityContainerSet,
		DynamoRabbitMQPriroritySet,
		wire.Struct(new(DynamoRabbitMQPriorityIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectMockedBackend(ctx context.Context, mockController *gomock.Controller) (*MockedBackend, error) {

	wire.Build(
		TestVersionConfiguratorSet,
		MockedRegistrySet,
		MockedPublihserSet,
		MockedUserNotificationBroker,
		routes.NewEngine,
		wire.Struct(new(MockedBackend), "*"),
	)

	return nil, nil
}

func InjectRabbitMQPriorityDeployer(envfile string) (*deployments.RabbitMQPriorityDeployer, func(), error) {

	wire.Build(
		EnvConfigSet,
		deployments.NewRabbitMQPriorityDeployer,
	)

	return nil, nil, nil
}

func InjectSQSPriorityDeployer(envfile string) (*deployments.SQSPriorityDeployer, func(), error) {

	wire.Build(
		EnvConfigSet,
		deployments.NewSQSPriorityDeployer,
	)

	return nil, nil, nil
}
