//go:build wireinject
// +build wireinject

package dependencyinjection

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/notifique/controllers"
	"github.com/notifique/routes"
	"github.com/redis/go-redis/v9"
	gomock "go.uber.org/mock/gomock"

	"github.com/notifique/internal"
	bk "github.com/notifique/internal/broker"
	cfg "github.com/notifique/internal/config"
	"github.com/notifique/internal/deployments"
	pub "github.com/notifique/internal/publish"
	dynamostorage "github.com/notifique/internal/storage/dynamodb"
	pg "github.com/notifique/internal/storage/postgres"
	tcfg "github.com/notifique/test/config"
	c "github.com/notifique/test/containers"
	mk "github.com/notifique/test/mocks"
)

type PostgresMockedPubIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	RedisContainer    *c.RedisContainer
	Storage           *pg.Storage
	Publisher         *mk.MockNotificationPublisher
	Broker            *bk.Redis
	Engine            *gin.Engine
}

type DynamoMockedPubIntegrationTest struct {
	DynamoContainer *c.DynamoContainer
	RedisContainer  *c.RedisContainer
	Storage         *dynamostorage.Storage
	Publisher       *mk.MockNotificationPublisher
	Broker          *bk.Redis
	Engine          *gin.Engine
}

type PgSQSPriorityIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	SQSContainer      *c.SQSPriorityContainer
	Storage           *pg.Storage
	Publisher         *pub.PriorityPublisher
}

type PgRabbitMQPriorityIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	Storage           *pg.Storage
	Publisher         *pub.PriorityPublisher
}

type DynamoSQSPriorityIntegrationTest struct {
	DynamoContainer *c.DynamoContainer
	SQSContainer    *c.SQSPriorityContainer
	Storage         *dynamostorage.Storage
	Publisher       *pub.PriorityPublisher
}

type DynamoRabbitMQPriorityIntegrationTest struct {
	DynamoContainer   *c.DynamoContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	Storage           *dynamostorage.Storage
	Publisher         *pub.PriorityPublisher
}

type MockedBackend struct {
	Storage   *mk.MockedStorage
	Publisher *mk.MockNotificationPublisher
	Broker    *mk.MockUserNotificationBroker
	Engine    *gin.Engine
}

var DynamoSet = wire.NewSet(
	dynamostorage.NewDynamoDBClient,
	dynamostorage.NewDynamoDBStorage,
	wire.Bind(new(dynamostorage.DynamoDBAPI), new(*dynamodb.Client)),
	wire.Bind(new(routes.Storage), new(*dynamostorage.Storage)),
	wire.Bind(new(controllers.NotificationStorage), new(*dynamostorage.Storage)),
)

var PostgresSet = wire.NewSet(
	pg.NewPostgresStorage,
	wire.Bind(new(routes.Storage), new(*pg.Storage)),
	wire.Bind(new(controllers.NotificationStorage), new(*pg.Storage)),
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
	internal.NewRedisClient,
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
	wire.Bind(new(dynamostorage.DynamoConfigurator), new(*c.DynamoContainer)),
)

var RedisContainerSet = wire.NewSet(
	c.NewRedisContainer,
	wire.Bind(new(internal.RedisConfigurator), new(*c.RedisContainer)),
	wire.Bind(new(bk.BrokerConfigurator), new(*c.RedisContainer)),
)

var MockedPublihserSet = wire.NewSet(
	mk.NewMockNotificationPublisher,
	wire.Bind(new(controllers.NotificationPublisher), new(*mk.MockNotificationPublisher)),
)

var MockedDistributionListStorageSet = wire.NewSet(
	mk.NewMockDistributionListStorage,
	wire.Bind(new(controllers.DistributionListStorage), new(*mk.MockDistributionListStorage)),
)

var MockedUserStorageSet = wire.NewSet(
	mk.NewMockUserStorage,
	wire.Bind(new(controllers.UserStorage), new(*mk.MockUserStorage)),
)

var MockedNotificationStorageSet = wire.NewSet(
	mk.NewMockNotificationStorage,
	wire.Bind(new(controllers.NotificationStorage), new(*mk.MockNotificationStorage)),
)

var MockedUserNotificationBroker = wire.NewSet(
	mk.NewMockUserNotificationBroker,
	wire.Bind(new(controllers.UserNotificationBroker), new(*mk.MockUserNotificationBroker)),
)

var MockedStorageSet = wire.NewSet(
	MockedDistributionListStorageSet,
	MockedUserStorageSet,
	MockedNotificationStorageSet,
	mk.NewMockedStorage,
	wire.Bind(new(routes.Storage), new(*mk.MockedStorage)),
)

var TestVersionConfiguratorSet = wire.NewSet(
	tcfg.NewTestVersionConfigurator,
	wire.Bind(new(routes.VersionConfigurator), new(tcfg.TestVersionConfiguratorFunc)),
)

var EnvConfigSet = wire.NewSet(
	cfg.NewEnvConfig,
	wire.Bind(new(pg.PostgresConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(dynamostorage.DynamoConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.SQSConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.RabbitMQConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.RabbitMQPriorityConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.SQSPriorityConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(internal.RedisConfigurator), new(*cfg.EnvConfig)),
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
		MockedStorageSet,
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
