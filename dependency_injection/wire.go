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

const (
	SQS_BASE_ENDPOINT       = "SQS_BASE_ENDPOINT"
	SQS_REGION              = "SQS_REGION"
	DYNAMO_BASE_ENDPOINT    = "DYNAMO_BASE_ENDPOINT"
	DYNAMO_REGION           = "DYNAMO_REGION"
	PRIORITY_SQS_LOW_URL    = "SQS_LOW_URL"
	PRIORITY_SQS_MEDIUM_URL = "SQS_MEDIUM_URL"
	PRIORITY_SQS_HIGH_URL   = "SQS_HIGH_URL"
	POSTGRES_URL            = "POSTGRES_URL"
)

type Storage interface {
	controllers.NotificationStorage
	controllers.UserStorage
	controllers.DistributionListStorage
}

type ConfigLoader interface {
	GetConfigValue(key string) (string, bool)
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
	pub.MakeSQSPublisher,
	wire.Bind(new(pub.SQSAPI), new(*sqs.Client)),
	wire.Bind(new(controllers.NotificationPublisher), new(*pub.SQSPublisher)),
)

var PostgresContainerSet = wire.NewSet(
	c.MakePostgresContainer,
	MakePostgresUrlFromContainer,
)

var SQSContainerSet = wire.NewSet(
	c.MakeSQSContainer,
	MakeSQSConfigFromContainer,
)

var DynamoContainerSet = wire.NewSet(
	c.MakeDynamoContainer,
	MakeDynamoConfigFromContainer,
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

func MakePostgresUrlFromContainer(container *c.PostgresContainer) (pg.PostgresURL, error) {

	if container == nil {
		return "", fmt.Errorf("postgres container is null")
	}

	return (pg.PostgresURL)(container.URI), nil
}

func MakeSQSConfigFromContainer(container *c.SQSContainer) (cfg pub.SQSConfig, err error) {

	if container == nil {
		return cfg, fmt.Errorf("sqs container is null")
	}

	clientCfg := pub.SQSClientConfig{BaseEndpoint: &container.URI}
	client, err := pub.MakeSQSClient(clientCfg)

	if err != nil {
		return cfg, fmt.Errorf("failed to create client - %w", err)
	}

	cfg.Client = client
	cfg.Urls = container.SQSEndpoints

	return
}

func MakeDynamoConfigFromContainer(container *c.DynamoContainer) (cfg ddb.DynamoClientConfig, err error) {

	if container == nil {
		return cfg, fmt.Errorf("sqs container is null")
	}

	cfg.BaseEndpoint = &container.URI

	return
}

func MakeSQSClient(cfg ConfigLoader) (pub.SQSAPI, error) {
	clientCfg := pub.SQSClientConfig{}

	if baseEndpoint, ok := cfg.GetConfigValue(SQS_BASE_ENDPOINT); ok {
		clientCfg.BaseEndpoint = &baseEndpoint
	}

	if region, ok := cfg.GetConfigValue(SQS_REGION); ok {
		clientCfg.Region = &region
	}

	return pub.MakeSQSClient(clientCfg)
}

func MakeSQSConfig(cfg ConfigLoader) (sqsCfg pub.SQSConfig, err error) {

	client, err := MakeSQSClient(cfg)

	if err != nil {
		return sqsCfg, err
	}

	urls := MakeSQSUrls(cfg)

	sqsCfg.Client = client
	sqsCfg.Urls = urls

	return
}

func MakeEngine(storage Storage, pub controllers.NotificationPublisher) *gin.Engine {

	r := gin.Default()

	routes.SetupNotificationRoutes(r, storage, pub)
	routes.SetupDistributionListRoutes(r, storage)
	routes.SetupUsersRoutes(r, storage)

	return r
}

func MakeSQSUrls(cfg ConfigLoader) (endpoints pub.SQSEndpoints) {

	low, _ := cfg.GetConfigValue(PRIORITY_SQS_LOW_URL)
	medium, _ := cfg.GetConfigValue(PRIORITY_SQS_MEDIUM_URL)
	high, _ := cfg.GetConfigValue(PRIORITY_SQS_HIGH_URL)

	endpoints.Low = &low
	endpoints.Medium = &medium
	endpoints.High = &high

	return
}

func GetPostgresUrl(cfg ConfigLoader) (pg.PostgresURL, error) {
	url, ok := cfg.GetConfigValue(POSTGRES_URL)

	if !ok {
		return "", fmt.Errorf("%s is not set", POSTGRES_URL)
	}

	return pg.PostgresURL(url), nil
}

func MakeDynamoClientConfig(cfg ConfigLoader) (dynamoCfg ddb.DynamoClientConfig) {

	clientCfg := ddb.DynamoClientConfig{}

	if baseEndpoint, ok := cfg.GetConfigValue(DYNAMO_BASE_ENDPOINT); ok {
		clientCfg.BaseEndpoint = &baseEndpoint
	}

	if region, ok := cfg.GetConfigValue(DYNAMO_REGION); ok {
		clientCfg.Region = &region
	}

	return
}

func InjectDynamoSQSEngine(cfg ConfigLoader) (*gin.Engine, error) {

	wire.Build(
		MakeSQSConfig,
		MakeDynamoClientConfig,
		DynamoSet,
		SQSSet,
		MakeEngine,
	)

	return nil, nil
}

func InjectPostgresSQSEngine(cfg ConfigLoader) (*gin.Engine, error) {

	wire.Build(
		MakeSQSConfig,
		GetPostgresUrl,
		PostgresSet,
		SQSSet,
		MakeEngine,
	)

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
