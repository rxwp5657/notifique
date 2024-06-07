//go:build wireinject
// +build wireinject

package dependencyinjection

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"github.com/notifique/controllers"
	pub "github.com/notifique/internal/publisher"
	ddb "github.com/notifique/internal/storage/dynamodb"
	pg "github.com/notifique/internal/storage/postgres"
	"github.com/notifique/routes"
	c "github.com/notifique/test/containers"
)

type Storage interface {
	controllers.NotificationStorage
	controllers.UserStorage
	controllers.DistributionListStorage
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

var SQSSet = wire.NewSet(
	pub.MakeSQSClient,
	MakeSQSConfig,
	pub.MakeSQSPublisher,
	wire.Bind(new(pub.SQSAPI), new(*sqs.Client)),
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.SQSPublisher)),
)

var PostgresContainerSet = wire.NewSet(
	c.MakePostgresContainer,
	MakePostgresUrl,
)

var SQSContainerSet = wire.NewSet(
	c.MakeSQSContainer,
	MakeSQSEndpoint,
	MakeSQSEndpoints,
)

var DynamoContainerSet = wire.NewSet(
	c.MakeDynamoContainer,
	MakeDynamoEndpoint,
)

type PostgresSQSIntegrationTest struct {
	PostgresContainer *c.PostgresContainer
	SQSContainer      *c.SQSContainer
	Storage           *pg.PostgresStorage
	Publisher         *pub.SQSPublisher
	Engine            *gin.Engine
}

func (app *PostgresSQSIntegrationTest) Cleanup() error {
	if err := app.PostgresContainer.CleanupFn(); err != nil {
		return fmt.Errorf("failed to cleanup postgres container - %w", err)
	}

	if err := app.SQSContainer.CleanupFn(); err != nil {
		return fmt.Errorf("failed to terminate sqs container - %w", err)
	}

	return nil
}

type DynamoSQSIntegrationTest struct {
	DynamoContainer *c.DynamoContainer
	SQSContainer    *c.SQSContainer
	DynamoDBStorage *ddb.DynamoDBStorage
	SQSPublisher    *pub.SQSPublisher
	Engine          *gin.Engine
}

func (app *DynamoSQSIntegrationTest) Cleanup() error {
	if err := app.DynamoContainer.CleanupFn(); err != nil {
		return fmt.Errorf("failed to cleanup dynamo container - %w", err)
	}

	if err := app.SQSContainer.CleanupFn(); err != nil {
		return fmt.Errorf("failed to terminate sqs container - %w", err)
	}

	return nil
}

func MakePostgresUrl(container *c.PostgresContainer) (pg.PostgresURL, error) {
	if container == nil {
		return "", fmt.Errorf("postgres container is null")
	}
	return (pg.PostgresURL)(container.URI), nil
}

func MakeSQSEndpoint(container *c.SQSContainer) (*pub.SQSEndpoint, error) {

	if container == nil {
		return nil, fmt.Errorf("sqs container is null")
	}

	return (*pub.SQSEndpoint)(&container.URI), nil
}

func MakeSQSEndpoints(container *c.SQSContainer) (pub.SQSEndpoints, error) {

	if container == nil {
		return pub.SQSEndpoints{}, fmt.Errorf("sqs container is null")
	}

	return container.SQSEndpoints, nil
}

func MakeDynamoEndpoint(container *c.DynamoContainer) (*ddb.DynamoEndpoint, error) {

	if container == nil {
		return nil, fmt.Errorf("sqs container is null")
	}

	return (*ddb.DynamoEndpoint)(&container.URI), nil
}

func MakeSQSConfig(client pub.SQSAPI, urls pub.SQSEndpoints) pub.SQSConfig {
	return pub.SQSConfig{
		Client: client,
		Urls:   urls,
	}
}

func MakeEngine(storage Storage, pub controllers.NotificationPublisher) *gin.Engine {

	r := gin.Default()

	routes.SetupNotificationRoutes(r, storage, pub)
	routes.SetupDistributionListRoutes(r, storage)
	routes.SetupUsersRoutes(r, storage)

	return r
}

func InjectDynamoSQSEngine(dynamoEndpoint *ddb.DynamoEndpoint, sqsEndpoint *pub.SQSEndpoint, urls pub.SQSEndpoints) (*gin.Engine, error) {
	wire.Build(wire.NewSet(DynamoSet, SQSSet, MakeEngine))
	return nil, nil
}

func InjectPostgresSQSEngine(postgresUrl pg.PostgresURL, sqsEndpoint *pub.SQSEndpoint, urls pub.SQSEndpoints) (*gin.Engine, error) {
	wire.Build(wire.NewSet(PostgresSet, SQSSet, MakeEngine))
	return nil, nil
}

func InjectPostgresSQSContainerTesting(ctx context.Context) (*PostgresSQSIntegrationTest, error) {
	wire.Build(
		PostgresContainerSet,
		SQSContainerSet,
		PostgresSet,
		SQSSet,
		MakeEngine,
		wire.Struct(new(PostgresSQSIntegrationTest), "*"),
	)

	return nil, nil
}

func InjectDynamoSQSContainerTesting(ctx context.Context) (*DynamoSQSIntegrationTest, error) {
	wire.Build(
		DynamoContainerSet,
		SQSContainerSet,
		DynamoSet,
		SQSSet,
		MakeEngine,
		wire.Struct(new(DynamoSQSIntegrationTest), "*"),
	)

	return nil, nil
}
