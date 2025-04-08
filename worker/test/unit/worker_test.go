package unit_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/di"
	"github.com/notifique/worker/internal/providers"
	"github.com/notifique/worker/internal/worker"
	"go.uber.org/mock/gomock"
)

func TestWorker(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	notificationChan := make(chan dto.NotificationMsg)
	defer close(notificationChan)

	ctx := context.Background()
	scenario := di.InjectMockedWorkerIntegrationTest(ctx, controller, notificationChan)

	template := dto.NotificationTemplateDetails{
		Id:               "template-id",
		Name:             "template-name",
		IsHtml:           true,
		ContentsTemplate: "Test Content",
		TitleTemplate:    "Test Title {{var1}}",
		Description:      "Test Description {{var2}}",
		Variables: []dto.TemplateVariable{
			{
				Name:     "var1",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "var2",
				Type:     "string",
				Required: true,
			},
		},
	}

	rawNotification := dto.NotificationMsg{
		DeleteTag: "123",
		Payload: dto.NotificationMsgPayload{
			Id:   "notification-1",
			Hash: "hash-1",
			NotificationReq: dto.NotificationReq{
				RawContents: &dto.RawContents{
					Title:    "Test Title",
					Contents: "Test Content",
				},
				Topic:      "test-topic",
				Recipients: []string{"user1"},
				Channels:   []dto.NotificationChannel{dto.InApp, dto.Email},
			},
		},
	}

	templateNotification := dto.NotificationMsg{
		DeleteTag: "123",
		Payload: dto.NotificationMsgPayload{
			Id:   "notification-2",
			Hash: "hash-2",
			NotificationReq: dto.NotificationReq{
				Topic:      "test-topic",
				Recipients: []string{"user1"},
				Channels:   []dto.NotificationChannel{dto.InApp, dto.Email},
				TemplateContents: &dto.TemplateContents{
					Id: "template-id",
					Variables: []dto.TemplateVariableContents{
						{
							Name:  "var1",
							Value: "Test Value 1",
						},
						{
							Name:  "var2",
							Value: "Test Value 2",
						},
					},
				},
			},
		},
	}

	testEmails := map[string]string{
		"user1": "user1@test.com",
		"user2": "user2@test.com",
		"user3": "user3@test.com",
	}

	tests := []struct {
		name      string
		msg       dto.NotificationMsg
		setupMock func(notification dto.NotificationMsg)
	}{
		{
			// Happy path for raw notification
			name: "successfully process notification with raw contents",
			msg:  rawNotification,
			setupMock: func(notification dto.NotificationMsg) {
				scenario.
					NotificationInfoProvider.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notification.Payload.Id).
					Return(dto.NotificationStatus(dto.Queued), nil).
					Times(1)

				scenario.
					NotificationInfoUpdater.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), dto.NotificationStatusLog{
						NotificationId: notification.Payload.Id,
						Status:         dto.Sending,
					}).
					Return(nil).
					Times(1)

				for _, recipient := range notification.Payload.Recipients {
					scenario.
						UserInfoProvider.
						EXPECT().
						GetUserInfo(gomock.Any(), recipient).
						Return(providers.UserInfo{
							UserId: recipient,
							Name:   "Test User",
							Email:  testEmails[recipient],
							Phone:  "1234567890",
						}, nil).
						Times(1)
				}

				scenario.
					NotificationInfoProvider.
					EXPECT().
					GetRecipientNotificationStatuses(gomock.Any(), providers.StatusFilters{
						NotificationId: notification.Payload.Id,
						Channels:       notification.Payload.Channels,
						Statuses:       []dto.NotificationStatus{dto.Sent},
					}).Return([]dto.RecipientNotificationStatus{}, nil).
					Times(1)

				expectedInAppRecipientStatusLogs := []dto.RecipientNotificationStatus{}
				expectedInAppNotification := make([]dto.UserNotificationReq, 0, len(notification.Payload.Recipients))

				for _, recipient := range notification.Payload.Recipients {
					expectedInAppNotification = append(expectedInAppNotification, dto.UserNotificationReq{
						UserId:   recipient,
						Title:    notification.Payload.RawContents.Title,
						Contents: notification.Payload.RawContents.Contents,
						Topic:    notification.Payload.Topic,
						Image:    notification.Payload.Image,
					})

					expectedInAppRecipientStatusLogs = append(expectedInAppRecipientStatusLogs, dto.RecipientNotificationStatus{
						UserId:  recipient,
						Status:  string(dto.Sent),
						Channel: string(dto.InApp),
					})
				}

				scenario.
					InAppSender.
					EXPECT().
					SendNotifications(gomock.Any(), expectedInAppNotification).
					Return(nil).
					Times(1)

				expectedEmailRecipientStatusLogs := []dto.RecipientNotificationStatus{}
				expectedEmailNotifications := make([]dto.UserEmailNotificationReq, 0, len(notification.Payload.Recipients))

				for _, recipient := range notification.Payload.Recipients {
					expectedEmailNotifications = append(expectedEmailNotifications, dto.UserEmailNotificationReq{
						Email:  testEmails[recipient],
						IsHtml: false,
						UserNotificationReq: dto.UserNotificationReq{
							UserId:   recipient,
							Title:    notification.Payload.RawContents.Title,
							Contents: notification.Payload.RawContents.Contents,
							Topic:    notification.Payload.Topic,
							Image:    notification.Payload.Image,
						},
					})

					expectedEmailRecipientStatusLogs = append(expectedEmailRecipientStatusLogs, dto.RecipientNotificationStatus{
						UserId:  recipient,
						Status:  string(dto.Sent),
						Channel: string(dto.Email),
					})
				}

				scenario.
					EmailSender.
					EXPECT().
					SendNotifications(gomock.Any(), expectedEmailNotifications).
					Return(nil).
					Times(1)

				scenario.
					NotificationInfoUpdater.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), dto.NotificationStatusLog{
						NotificationId: notification.Payload.Id,
						Status:         dto.Sent,
					}).
					Return(nil).
					Times(1)

				statusLogs := []dto.RecipientNotificationStatus{}
				statusLogs = append(statusLogs, expectedInAppRecipientStatusLogs...)
				statusLogs = append(statusLogs, expectedEmailRecipientStatusLogs...)

				scenario.
					NotificationInfoUpdater.
					EXPECT().
					UpdateRecipientNotificationStatus(gomock.Any(), notification.Payload.Id, statusLogs).
					Return(nil).
					Times(1)

				scenario.
					QueueConsumer.EXPECT().
					Ack(gomock.Any(), notification.DeleteTag).
					Return(nil).
					Times(1)
			},
		},
		{
			// Happy path for template notification
			name: "successfully process notification with template contents",
			msg:  templateNotification,
			setupMock: func(notification dto.NotificationMsg) {
				scenario.
					NotificationInfoProvider.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notification.Payload.Id).
					Return(dto.NotificationStatus(dto.Queued), nil).
					Times(1)

				scenario.
					NotificationInfoUpdater.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), dto.NotificationStatusLog{
						NotificationId: notification.Payload.Id,
						Status:         dto.Sending,
					}).
					Return(nil).
					Times(1)

				for _, recipient := range notification.Payload.Recipients {
					scenario.
						UserInfoProvider.
						EXPECT().
						GetUserInfo(gomock.Any(), recipient).
						Return(providers.UserInfo{
							UserId: recipient,
							Name:   "Test User",
							Email:  testEmails[recipient],
							Phone:  "1234567890",
						}, nil).
						Times(1)
				}

				scenario.
					NotificationInfoProvider.
					EXPECT().
					GetRecipientNotificationStatuses(gomock.Any(), providers.StatusFilters{
						NotificationId: notification.Payload.Id,
						Channels:       notification.Payload.Channels,
						Statuses:       []dto.NotificationStatus{dto.Sent},
					}).Return([]dto.RecipientNotificationStatus{}, nil).
					Times(1)

				scenario.
					NotificationInfoProvider.
					EXPECT().
					GetNotificationTemplate(gomock.Any(), notification.Payload.TemplateContents.Id).
					Return(template, nil).
					Times(1)

				contents := buildTemplateNotification(notification.Payload, &template)

				expectedInAppRecipientStatusLogs := []dto.RecipientNotificationStatus{}
				expectedInAppNotification := make([]dto.UserNotificationReq, 0, len(notification.Payload.Recipients))

				for _, recipient := range notification.Payload.Recipients {
					expectedInAppNotification = append(expectedInAppNotification, dto.UserNotificationReq{
						UserId:   recipient,
						Title:    contents.Title,
						Contents: contents.Contents,
						Topic:    notification.Payload.Topic,
						Image:    notification.Payload.Image,
					})

					expectedInAppRecipientStatusLogs = append(expectedInAppRecipientStatusLogs, dto.RecipientNotificationStatus{
						UserId:  recipient,
						Status:  string(dto.Sent),
						Channel: string(dto.InApp),
					})
				}

				scenario.
					InAppSender.
					EXPECT().
					SendNotifications(gomock.Any(), expectedInAppNotification).
					Return(nil).
					Times(1)

				expectedEmailRecipientStatusLogs := []dto.RecipientNotificationStatus{}
				expectedEmailNotifications := make([]dto.UserEmailNotificationReq, 0, len(notification.Payload.Recipients))

				for _, recipient := range notification.Payload.Recipients {
					expectedEmailNotifications = append(expectedEmailNotifications, dto.UserEmailNotificationReq{
						Email:  testEmails[recipient],
						IsHtml: contents.IsHTML,
						UserNotificationReq: dto.UserNotificationReq{
							UserId:   recipient,
							Title:    contents.Title,
							Contents: contents.Contents,
							Topic:    notification.Payload.Topic,
							Image:    notification.Payload.Image,
						},
					})

					expectedEmailRecipientStatusLogs = append(expectedEmailRecipientStatusLogs, dto.RecipientNotificationStatus{
						UserId:  recipient,
						Status:  string(dto.Sent),
						Channel: string(dto.Email),
					})
				}

				scenario.
					EmailSender.
					EXPECT().
					SendNotifications(gomock.Any(), expectedEmailNotifications).
					Return(nil).
					Times(1)

				scenario.
					NotificationInfoUpdater.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), dto.NotificationStatusLog{
						NotificationId: notification.Payload.Id,
						Status:         dto.Sent,
					}).
					Return(nil).
					Times(1)

				statusLogs := []dto.RecipientNotificationStatus{}
				statusLogs = append(statusLogs, expectedInAppRecipientStatusLogs...)
				statusLogs = append(statusLogs, expectedEmailRecipientStatusLogs...)

				scenario.
					NotificationInfoUpdater.
					EXPECT().
					UpdateRecipientNotificationStatus(gomock.Any(), notification.Payload.Id, statusLogs).
					Return(nil).
					Times(1)

				scenario.
					QueueConsumer.EXPECT().
					Ack(gomock.Any(), notification.DeleteTag).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "notification was cancelled",
			msg:  rawNotification,
			setupMock: func(notification dto.NotificationMsg) {
				scenario.
					NotificationInfoProvider.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notification.Payload.Id).
					Return(dto.NotificationStatus(dto.Canceled), nil).
					Times(1)

				scenario.
					QueueConsumer.EXPECT().
					Ack(gomock.Any(), notification.DeleteTag).
					Return(nil).
					Times(1)
			},
		},
		{
			name: "all messages have been sent but wasn't acknowledged",
			msg:  rawNotification,
			setupMock: func(notification dto.NotificationMsg) {
				scenario.
					NotificationInfoProvider.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notification.Payload.Id).
					Return(dto.NotificationStatus(dto.Failed), nil).
					Times(1)

				scenario.
					NotificationInfoUpdater.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), dto.NotificationStatusLog{
						NotificationId: notification.Payload.Id,
						Status:         dto.Sending,
					}).
					Return(nil).
					Times(1)

				statuses := []dto.RecipientNotificationStatus{}

				for _, recipient := range notification.Payload.Recipients {
					statuses = append(statuses, dto.RecipientNotificationStatus{
						UserId:  recipient,
						Status:  string(dto.Sent),
						Channel: string(dto.InApp),
					})
				}

				scenario.
					NotificationInfoProvider.
					EXPECT().
					GetRecipientNotificationStatuses(gomock.Any(), providers.StatusFilters{
						NotificationId: notification.Payload.Id,
						Channels:       notification.Payload.Channels,
						Statuses:       []dto.NotificationStatus{dto.Sent},
					}).Return(statuses, nil).
					Times(1)

				scenario.
					NotificationInfoUpdater.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), dto.NotificationStatusLog{
						NotificationId: notification.Payload.Id,
						Status:         dto.Sent,
					}).
					Return(nil).
					Times(1)

				scenario.
					QueueConsumer.EXPECT().
					Ack(gomock.Any(), notification.DeleteTag).
					Return(nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(tt.msg)
			scenario.Worker.ProcessNotification(ctx, tt.msg)
		})
	}
}

func buildTemplateNotification(p dto.NotificationMsgPayload, t *dto.NotificationTemplateDetails) worker.NotificationContents {

	contents := worker.NotificationContents{
		Topic:  p.Topic,
		Image:  p.Image,
		IsHTML: t.IsHtml,
	}

	contents.Title = t.TitleTemplate
	contents.Contents = t.ContentsTemplate

	for _, value := range p.TemplateContents.Variables {
		name := fmt.Sprintf("{{%s}}", value.Name)
		contents.Title = strings.ReplaceAll(contents.Title, name, value.Value)
		contents.Contents = strings.ReplaceAll(contents.Contents, name, value.Value)
	}

	return contents
}
