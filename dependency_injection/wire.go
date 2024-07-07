//go:build wireinject
// +build wireinject

package dependencyinjection

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/wire"
	"github.com/notifique/controllers"
	"github.com/notifique/routes"
	"github.com/redis/go-redis/v9"

	"github.com/notifique/internal"
	bk "github.com/notifique/internal/broker"
	cfg "github.com/notifique/internal/config"
	"github.com/notifique/internal/deployments"
	pub "github.com/notifique/internal/publisher"
	ddb "github.com/notifique/internal/storage/dynamodb"
	pg "github.com/notifique/internal/storage/postgres"
	c "github.com/notifique/test/containers"
	mk "github.com/notifique/test/mocks"
)

type Storage interface {
	controllers.NotificationStorage
	controllers.UserStorage
	controllers.DistributionListStorage
}

type PostgresMockedPubIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	RedisContainer    *c.RedisContainer
	Storage           *pg.PostgresStorage
	Publisher         *mk.MockNotificationPublisher
	Broker            *bk.RedisBroker
	Engine            *gin.Engine
}

type DynamoMockedPubIntegrationTest struct {
	DynamoContainer *c.DynamoContainer
	RedisContainer  *c.RedisContainer
	Storage         *ddb.DynamoDBStorage
	Publisher       *mk.MockNotificationPublisher
	Broker          *bk.RedisBroker
	Engine          *gin.Engine
}

type PgSQSPriorityIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	SQSContainer      *c.SQSPriorityContainer
	Storage           *pg.PostgresStorage
	Publisher         *pub.PriorityPublisher
}

type PgRabbitMQPriorityIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	Storage           *pg.PostgresStorage
	Publisher         *pub.PriorityPublisher
}

type DynamoSQSPriorityIntegrationTest struct {
	DynamoContainer *c.DynamoContainer
	SQSContainer    *c.SQSPriorityContainer
	Storage         *ddb.DynamoDBStorage
	Publisher       *pub.PriorityPublisher
}

type DynamoRabbitMQPriorityIntegrationTest struct {
	DynamoContainer   *c.DynamoContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	Storage           *ddb.DynamoDBStorage
	Publisher         *pub.PriorityPublisher
}

var DynamoSet = wire.NewSet(
	ddb.NewDynamoDBClient,
	ddb.NewDynamoDBStorage,
	wire.Bind(new(ddb.DynamoDBAPI), new(*dynamodb.Client)),
	wire.Bind(new(Storage), new(*ddb.DynamoDBStorage)),
	wire.Bind(new(controllers.NotificationStorage), new(*ddb.DynamoDBStorage)),
)

var PostgresSet = wire.NewSet(
	pg.NewPostgresStorage,
	wire.Bind(new(Storage), new(*pg.PostgresStorage)),
	wire.Bind(new(controllers.NotificationStorage), new(*pg.PostgresStorage)),
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
	wire.Bind(new(controllers.UserNotificationBroker), new(*bk.RedisBroker)),
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
	wire.Bind(new(ddb.DynamoConfigurator), new(*c.DynamoContainer)),
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

var EnvConfigSet = wire.NewSet(
	cfg.NewEnvConfig,
	wire.Bind(new(pg.PostgresConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(ddb.DynamoConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.SQSConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.RabbitMQConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.RabbitMQPriorityConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(pub.SQSPriorityConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(internal.RedisConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(bk.BrokerConfigurator), new(*cfg.EnvConfig)),
)

func NewEngine(storage Storage, pub controllers.NotificationPublisher, bk controllers.UserNotificationBroker) *gin.Engine {

	r := gin.Default()

	routes.SetupNotificationRoutes(r, storage, pub)
	routes.SetupDistributionListRoutes(r, storage)
	routes.SetupUsersRoutes(r, storage, bk)

	return r
}

func InjectPgPrioritySQS(envfile string) (*gin.Engine, error) {

	wire.Build(
		EnvConfigSet,
		PostgresSQSPriroritySet,
		RedisUserNotificationBrokerSet,
		NewEngine,
	)

	return nil, nil
}

func InjectPgPriorityRabbitMQ(envfile string) (*gin.Engine, func(), error) {

	wire.Build(
		EnvConfigSet,
		PostgresRabbitMQPriroritySet,
		RedisUserNotificationBrokerSet,
		NewEngine,
	)

	return nil, nil, nil
}

func InjectDynamoPrioritySQS(envfile string) (*gin.Engine, error) {

	wire.Build(
		EnvConfigSet,
		DynamoSQSPriroritySet,
		RedisUserNotificationBrokerSet,
		NewEngine,
	)

	return nil, nil
}

func InjectDynamoPriorityRabbitMQ(envfile string) (*gin.Engine, func(), error) {

	wire.Build(
		EnvConfigSet,
		DynamoRabbitMQPriroritySet,
		RedisUserNotificationBrokerSet,
		NewEngine,
	)

	return nil, nil, nil
}

func InjectPgMockedPubIntegrationTest(ctx context.Context, mockController *gomock.Controller) (*PostgresMockedPubIntegrationTest, func(), error) {

	wire.Build(
		PostgresContainerSet,
		RedisContainerSet,
		PostgresSet,
		MockedPublihserSet,
		RedisUserNotificationBrokerSet,
		NewEngine,
		wire.Struct(new(PostgresMockedPubIntegrationTest), "*"),
	)

	return nil, nil, nil
}

func InjectDynamoMockedPubIntegrationTest(ctx context.Context, mockController *gomock.Controller) (*DynamoMockedPubIntegrationTest, func(), error) {

	wire.Build(
		DynamoContainerSet,
		RedisContainerSet,
		DynamoSet,
		MockedPublihserSet,
		RedisUserNotificationBrokerSet,
		NewEngine,
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
