//go:build wireinject
// +build wireinject

package di

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/wire"
	"github.com/notifique/shared/cache"
	"github.com/notifique/shared/clients"
	"github.com/notifique/shared/dto"
	wc "github.com/notifique/worker/internal/clients"
	cfg "github.com/notifique/worker/internal/config"
	"github.com/notifique/worker/internal/consumers"
	"github.com/notifique/worker/internal/providers"
	"github.com/notifique/worker/internal/sender"
	consumers_test "github.com/notifique/worker/internal/testutils/consumers"
	containers_test "github.com/notifique/worker/internal/testutils/containers"
	"github.com/notifique/worker/internal/testutils/mocks"
	"github.com/notifique/worker/internal/worker"
	redis "github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"
)

func ProvideSQSQueueCfg(c consumers.SQSQueueConfigurator) (consumers.SQSQueueCfg, error) {
	cfg, err := c.GetSQSQueueCfg()

	if err != nil {
		return consumers.SQSQueueCfg{}, fmt.Errorf("failed to get SQS queue config - %w", err)
	}

	return cfg, nil
}

func ProvideRabbitMQQueue(c consumers.RabbitMQConfigurator) (consumers.RabbitMQQueue, error) {
	queue, err := c.GetQueue()

	if err != nil {
		return "", fmt.Errorf("failed to get RabbitMQ queue - %w", err)
	}

	return queue, nil
}

func ProvideNotificationServiceClientConfigurator(c wc.NotificationServiceClientConfigurator) (*wc.NotificationServiceClientCfg, error) {
	cfg, err := c.GetNotificationServiceClientCfg()

	if err != nil {
		return &cfg, fmt.Errorf("failed to get Notification Service client config - %w", err)
	}

	return &cfg, nil
}

func ProvideSMTPConfigurator(c sender.SMTPConfigurator) (sender.SMTPConfig, error) {
	cfg, err := c.GetSMTPConfig()

	if err != nil {
		return sender.SMTPConfig{}, fmt.Errorf("failed to get SMTP config - %w", err)
	}

	return cfg, nil
}

func ProviderUserPoolId(c providers.CognitoUserInfoConfigurator) (providers.UserPoolID, error) {
	userPoolId, err := c.GetUserPoolId()

	if err != nil {
		return "", fmt.Errorf("failed to get User Pool ID - %w", err)
	}

	return userPoolId, nil
}

func ProvideNotificationMsgChanWriter(notificationChan chan dto.NotificationMsg) (chan<- dto.NotificationMsg, error) {
	return notificationChan, nil
}

func ProvideNotificationMsgChanReader(notificationChan chan dto.NotificationMsg) (<-chan dto.NotificationMsg, error) {
	return notificationChan, nil
}

var MockedUserInfoProviderSet = wire.NewSet(
	mocks.NewMockUserInfoProvider,
	wire.Bind(new(worker.UserInfoProvider), new(*mocks.MockUserInfoProvider)),
)

var MockedNotificationInfoProviderSet = wire.NewSet(
	mocks.NewMockNotificationInfoProvider,
	wire.Bind(new(worker.NotificationInfoProvider), new(*mocks.MockNotificationInfoProvider)),
)

var MockedNotificationInfoUpdaterSet = wire.NewSet(
	mocks.NewMockNotificationInfoUpdater,
	wire.Bind(new(worker.NotificationInfoUpdater), new(*mocks.MockNotificationInfoUpdater)),
)

var MockedQueueConsumerSet = wire.NewSet(
	mocks.NewMockQueueConsumer,
	wire.Bind(new(worker.QueueConsumer), new(*mocks.MockQueueConsumer)),
)

var MockedInAppSenderSet = wire.NewSet(
	mocks.NewMockInAppSender,
	wire.Bind(new(worker.InAppSender), new(*mocks.MockInAppSender)),
)

var MockedEmailSenderSet = wire.NewSet(
	mocks.NewMockEmailSender,
	wire.Bind(new(worker.EmailSender), new(*mocks.MockEmailSender)),
)

type MockedWorkerScenario struct {
	NotificationInfoProvider *mocks.MockNotificationInfoProvider
	NotificationInfoUpdater  *mocks.MockNotificationInfoUpdater
	UserInfoProvider         *mocks.MockUserInfoProvider
	QueueConsumer            *mocks.MockQueueConsumer
	InAppSender              *mocks.MockInAppSender
	EmailSender              *mocks.MockEmailSender
	Worker                   *worker.Worker
}

var SQSConsumerSet = wire.NewSet(
	ProvideSQSQueueCfg,
	clients.NewSQSClient,
	wire.Struct(new(consumers.SQSCfg), "*"),
	consumers.NewSQSConsumer,
	wire.Bind(new(consumers.SQSAPI), new(*sqs.Client)),
	wire.Bind(new(worker.QueueConsumer), new(*consumers.SQS)),
)

var RabbitMQConsumerSet = wire.NewSet(
	ProvideRabbitMQQueue,
	clients.NewRabbitMQClient,
	wire.Struct(new(consumers.RabbitMQCfg), "*"),
	consumers.NewRabbitMQConsumer,
	wire.Bind(new(consumers.RabbitMQAPI), new(*clients.RabbitMQ)),
	wire.Bind(new(worker.QueueConsumer), new(*consumers.RabbitMQ)),
)

var CognitoAuthProviderSet = wire.NewSet(
	wc.NewCognitoAuthProvider,
	wire.Bind(new(wc.AuthProvider), new(*wc.CognitoAuthProvider)),
)

var NotificationServiceClientSet = wire.NewSet(
	ProvideNotificationServiceClientConfigurator,
	wire.FieldsOf(new(*wc.NotificationServiceClientCfg), "NotificationServiceUrl"),
	wire.FieldsOf(new(*wc.NotificationServiceClientCfg), "NumRetries"),
	wire.FieldsOf(new(*wc.NotificationServiceClientCfg), "BaseDelay"),
	wire.FieldsOf(new(*wc.NotificationServiceClientCfg), "MaxDelay"),
	wire.Struct(new(wc.NotificationServiceClient), "*"),
)

var NotificationServiceProviderSet = wire.NewSet(
	providers.NewNotificationServiceProvider,
	wire.Bind(new(worker.NotificationInfoProvider), new(*providers.NotificationServiceProvider)),
)

var NotificationServiceSenderSet = wire.NewSet(
	sender.NewNotificationServiceSender,
	wire.Bind(new(worker.NotificationInfoUpdater), new(*sender.NotificationServiceSender)),
	wire.Bind(new(worker.InAppSender), new(*sender.NotificationServiceSender)),
)

var SMTPSenderSet = wire.NewSet(
	ProvideSMTPConfigurator,
	sender.NewSMTP,
	wire.Bind(new(worker.EmailSender), new(*sender.SMTP)),
)

var CognitoUserInfoProviderSet = wire.NewSet(
	providers.NewCognitoIdentityProvider,
	ProviderUserPoolId,
	wire.Struct(new(providers.CognitoUserInfoCfg), "*"),
	providers.NewCognitoUserInfoProvider,
	wire.Bind(new(worker.UserInfoProvider), new(*providers.CognitoUserInfo)),
)

var RedisSet = wire.NewSet(
	cache.NewRedisClient,
	wire.Bind(new(cache.CacheRedisApi), new(*redis.Client)),
)

var RedisCacheSet = wire.NewSet(
	cache.NewRedisCache,
	wire.Bind(new(cache.Cache), new(*cache.Redis)),
)

var EnvConfigSet = wire.NewSet(
	cfg.NewEnvConfig,
	wire.Bind(new(wc.CognitoAuthConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(wc.NotificationServiceClientConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(providers.CognitoIdentityProviderConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(providers.CognitoUserInfoConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(sender.SMTPConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(cache.RedisConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(clients.SQSConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(clients.RabbitMQConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(consumers.RabbitMQConfigurator), new(*cfg.EnvConfig)),
	wire.Bind(new(consumers.SQSQueueConfigurator), new(*cfg.EnvConfig)),
)

func InjectRabbitMQConsumerIntegrationTest(ctx context.Context, notificationChan chan<- dto.NotificationMsg) (*consumers_test.RabbitMQ, func(), error) {

	wire.Build(
		containers_test.NewRabbitMQConsumerContainer,
		wire.FieldsOf(new(*containers_test.RabbitMQConsumerContainer), "Client"),
		wire.FieldsOf(new(*containers_test.RabbitMQConsumerContainer), "Queue"),
		wire.Bind(new(consumers.RabbitMQAPI), new(*clients.RabbitMQ)),
		wire.Struct(new(consumers.RabbitMQCfg), "*"),
		wire.Struct(new(consumers_test.RabbitMQCfg), "*"),
		consumers_test.NewRabbitMQConsumerTest,
	)

	return nil, nil, nil
}

func InjectSQSConsumerIntegrationTest(ctx context.Context, notificationChan chan<- dto.NotificationMsg) (*consumers_test.SQS, func(), error) {

	wire.Build(
		containers_test.NewSQSConsumerContainer,
		wire.FieldsOf(new(*containers_test.SQSConsumerContainer), "Client"),
		wire.FieldsOf(new(*containers_test.SQSConsumerContainer), "QueueCfg"),
		wire.Bind(new(consumers.SQSAPI), new(*sqs.Client)),
		wire.Struct(new(consumers.SQSCfg), "*"),
		wire.Struct(new(consumers_test.SQSCfg), "*"),
		consumers_test.NewSQSConsumerTest,
	)

	return nil, nil, nil
}

func InjectMockedWorkerIntegrationTest(ctx context.Context, mockController *gomock.Controller, notificationChan <-chan dto.NotificationMsg) *MockedWorkerScenario {

	wire.Build(
		MockedUserInfoProviderSet,
		MockedNotificationInfoProviderSet,
		MockedNotificationInfoUpdaterSet,
		MockedQueueConsumerSet,
		MockedInAppSenderSet,
		MockedEmailSenderSet,
		wire.Struct(new(worker.WorkerCfg), "*"),
		worker.NewWorker,
		wire.Struct(new(MockedWorkerScenario), "*"),
	)

	return nil
}

func InjectRabbitMQWorker(ctx context.Context, envfile *string, notificationChan chan dto.NotificationMsg) (*worker.Worker, func(), error) {

	wire.Build(
		EnvConfigSet,
		ProvideNotificationMsgChanWriter,
		ProvideNotificationMsgChanReader,
		RedisSet,
		RedisCacheSet,
		RabbitMQConsumerSet,
		CognitoAuthProviderSet,
		NotificationServiceClientSet,
		NotificationServiceProviderSet,
		NotificationServiceSenderSet,
		SMTPSenderSet,
		CognitoUserInfoProviderSet,
		wire.Struct(new(worker.WorkerCfg), "*"),
		worker.NewWorker,
	)

	return nil, nil, nil
}

func InjectSQSWorker(ctx context.Context, envfile *string, notificationChan chan dto.NotificationMsg) (*worker.Worker, func(), error) {

	wire.Build(
		EnvConfigSet,
		ProvideNotificationMsgChanWriter,
		ProvideNotificationMsgChanReader,
		RedisSet,
		RedisCacheSet,
		SQSConsumerSet,
		CognitoAuthProviderSet,
		NotificationServiceClientSet,
		NotificationServiceProviderSet,
		NotificationServiceSenderSet,
		SMTPSenderSet,
		CognitoUserInfoProviderSet,
		wire.Struct(new(worker.WorkerCfg), "*"),
		worker.NewWorker,
	)

	return nil, nil, nil
}
