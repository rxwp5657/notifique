//go:build wireinject
// +build wireinject

package di

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/wire"
	"github.com/notifique/shared/clients"
	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/consumers"
	consumers_test "github.com/notifique/worker/internal/testutils/consumers"
	containers_test "github.com/notifique/worker/internal/testutils/containers"
	"github.com/notifique/worker/internal/testutils/mocks"
	"github.com/notifique/worker/internal/worker"
	"go.uber.org/mock/gomock"
)

var RabbitMQConsumerSet = wire.NewSet(
	consumers.NewRabbitMQConsumer,
)

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
