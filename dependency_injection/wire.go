package dependencyinjection

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/notifique/controllers"
	"github.com/notifique/routes"

	pub "github.com/notifique/internal/publisher"
	ddb "github.com/notifique/internal/storage/dynamodb"
	pg "github.com/notifique/internal/storage/postgres"
	c "github.com/notifique/test/containers"
)

type Storage interface {
	controllers.NotificationStorage
	controllers.UserStorage
	controllers.DistributionListStorage
}

type PgPrioritySQSConfigurator interface {
	pg.PostgresConfigurator
	pub.SQSConfigurator
	pub.PriorityQueueConfigurator
}

type PgPriorityRabbitMQConfigurator interface {
	pg.PostgresConfigurator
	pub.RabbitMQConfigurator
	pub.PriorityQueueConfigurator
}

type DynamoPrioritySQSConfigurator interface {
	ddb.DynamoConfigurator
	pub.SQSConfigurator
	pub.PriorityQueueConfigurator
}

type DynamoPriorityRabbitMQConfigurator interface {
	ddb.DynamoConfigurator
	pub.RabbitMQConfigurator
	pub.PriorityQueueConfigurator
}

type PostgresPrioritySQSIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	SQSContainer      *c.SQSPriorityContainer
	Storage           *pg.PostgresStorage
	Publisher         *pub.SQSPublisher
	Engine            *gin.Engine
}

type DynamoPrioritySQSIntegrationTest struct {
	DynamoContainer *c.DynamoContainer
	SQSContainer    *c.SQSPriorityContainer
	Storage         *ddb.DynamoDBStorage
	Publisher       *pub.SQSPublisher
	Engine          *gin.Engine
}

type PostgresPriorityRabbitMQIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	RabbitMQClient    *pub.RabbitMQClient
	Storage           *pg.PostgresStorage
	Publisher         *pub.RabbitMQPriorityPublisher
	Engine            *gin.Engine
}

type DynamoPriorityRabbitMQIntegrationTest struct {
	DynamoContainer   *c.DynamoContainer
	RabbitMQContainer *c.RabbitMQPriorityContainer
	RabbitMQClient    *pub.RabbitMQClient
	Storage           *ddb.DynamoDBStorage
	Publisher         *pub.RabbitMQPriorityPublisher
	Engine            *gin.Engine
}

func (it *PostgresPrioritySQSIntegrationTest) Cleanup() error {

	if err := it.PostgresContainer.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup postgres container - %w", err)
	}

	if err := it.SQSContainer.Container.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup sqs container - %w", err)
	}

	return nil
}

func (it *DynamoPrioritySQSIntegrationTest) Cleanup() error {

	if err := it.DynamoContainer.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup dynamo container - %w", err)
	}

	if err := it.SQSContainer.Container.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup sqs container - %w", err)
	}

	return nil
}

func (it *PostgresPriorityRabbitMQIntegrationTest) Cleanup() error {

	if err := it.PostgresContainer.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup postgres container - %w", err)
	}

	if err := it.RabbitMQClient.Close(); err != nil {
		return fmt.Errorf("failed to close rabbitmq connection - %w", err)
	}

	if err := it.RabbitMQContainer.Container.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup rabbitmq container - %w", err)
	}

	return nil
}

func (it *DynamoPriorityRabbitMQIntegrationTest) Cleanup() error {

	if err := it.DynamoContainer.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup dynamo container - %w", err)
	}

	if err := it.RabbitMQClient.Close(); err != nil {
		return fmt.Errorf("failed to close rabbitmq connection - %w", err)
	}

	if err := it.RabbitMQContainer.Container.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup rabbitmq container - %w", err)
	}

	return nil
}

var DynamoSet = wire.NewSet(
	ddb.MakeDynamoDBClient,
	ddb.MakeDynamoDBStorage,
	wire.Bind(new(ddb.DynamoDBAPI), new(*dynamodb.Client)),
	wire.Bind(new(Storage), new(*ddb.DynamoDBStorage)),
)

var PostgresSet = wire.NewSet(
	pg.MakePostgresStorage,
	wire.Bind(new(Storage), new(*pg.PostgresStorage)),
)

var SQSPrioritySet = wire.NewSet(
	pub.MakeSQSClient,
	pub.MakeSQSPublisher,
	wire.Bind(new(pub.SQSAPI), new(*sqs.Client)),
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.SQSPublisher)),
)

var RabbitMQPrioritySet = wire.NewSet(
	pub.MakeRabbitMQClient,
	pub.MakeRabbitMQPriorityPub,
	wire.Bind(new(pub.RabbitMQAPI), new(*pub.RabbitMQClient)),
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.RabbitMQPriorityPublisher)),
)

var PostgresContainerSet = wire.NewSet(
	c.MakePostgresContainer,
	wire.Bind(new(pg.PostgresConfigurator), new(*c.PostgresContainer)),
)

var SQSPriorityContainerSet = wire.NewSet(
	c.MakeSQSPriorityContainer,
	wire.Bind(new(pub.SQSConfigurator), new(*c.SQSPriorityContainer)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*c.SQSPriorityContainer)),
)

var RabbitMQPriorityContainerSet = wire.NewSet(
	c.MakeRabbitMQPriorityContainer,
	wire.Bind(new(pub.RabbitMQConfigurator), new(*c.RabbitMQPriorityContainer)),
	wire.Bind(new(pub.PriorityQueueConfigurator), new(*c.RabbitMQPriorityContainer)),
)

var DynamoContainerSet = wire.NewSet(
	c.MakeDynamoContainer,
	wire.Bind(new(ddb.DynamoConfigurator), new(*c.DynamoContainer)),
)

func MakeEngine(storage Storage, pub controllers.NotificationPublisher) *gin.Engine {

	r := gin.Default()

	routes.SetupNotificationRoutes(r, storage, pub)
	routes.SetupDistributionListRoutes(r, storage)
	routes.SetupUsersRoutes(r, storage)

	return r
}

func InjectPgPrioritySQS(pgc pg.PostgresConfigurator, sc pub.SQSConfigurator, qc pub.PriorityQueueConfigurator) (*gin.Engine, error) {

	wire.Build(
		PostgresSet,
		SQSPrioritySet,
		MakeEngine,
	)

	return nil, nil
}

func InjectPgPriorityRabbitMQ(pgc pg.PostgresConfigurator, rc pub.RabbitMQConfigurator, qc pub.PriorityQueueConfigurator) (*gin.Engine, error) {

	wire.Build(
		PostgresSet,
		RabbitMQPrioritySet,
		MakeEngine,
	)

	return nil, nil
}

func InjectDynamoPrioritySQS(dc ddb.DynamoConfigurator, sc pub.SQSConfigurator, qc pub.PriorityQueueConfigurator) (*gin.Engine, error) {

	wire.Build(
		DynamoSet,
		SQSPrioritySet,
		MakeEngine,
	)

	return nil, nil
}

func InjectDynamoPriorityRabbitMQ(dc ddb.DynamoConfigurator, rc pub.RabbitMQConfigurator, qc pub.PriorityQueueConfigurator) (*gin.Engine, error) {

	wire.Build(
		DynamoSet,
		RabbitMQPrioritySet,
		MakeEngine,
	)

	return nil, nil
}

func InjectPgPrioritySQSIntegrationTest(ctx context.Context) (*PostgresPrioritySQSIntegrationTest, error) {

	wire.Build(
		PostgresContainerSet,
		SQSPriorityContainerSet,
		PostgresSet,
		SQSPrioritySet,
		MakeEngine,
		wire.Struct(new(PostgresPrioritySQSIntegrationTest), "*"),
	)

	return nil, nil
}

func InjectPgPriorityRabbitMQIntegrationTest(ctx context.Context) (*PostgresPriorityRabbitMQIntegrationTest, error) {

	wire.Build(
		PostgresContainerSet,
		RabbitMQPriorityContainerSet,
		PostgresSet,
		RabbitMQPrioritySet,
		MakeEngine,
		wire.Struct(new(PostgresPriorityRabbitMQIntegrationTest), "*"),
	)

	return nil, nil
}

func InjectDynamoPrioritySQSIntegrationTest(ctx context.Context) (*DynamoPrioritySQSIntegrationTest, error) {

	wire.Build(
		DynamoContainerSet,
		SQSPriorityContainerSet,
		DynamoSet,
		SQSPrioritySet,
		MakeEngine,
		wire.Struct(new(DynamoPrioritySQSIntegrationTest), "*"),
	)

	return nil, nil
}

func InjectDynamoPriorityRabbitMQIntegrationTest(ctx context.Context) (*DynamoPriorityRabbitMQIntegrationTest, error) {

	wire.Build(
		DynamoContainerSet,
		RabbitMQPriorityContainerSet,
		DynamoSet,
		RabbitMQPrioritySet,
		MakeEngine,
		wire.Struct(new(DynamoPriorityRabbitMQIntegrationTest), "*"),
	)

	return nil, nil
}
