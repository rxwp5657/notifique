package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/providers"
)

type UserInfoProvider interface {
	GetUserInfo(ctx context.Context, userID string) (providers.UserInfo, error)
}

type NotificationInfoProvider interface {
	GetNotificationStatus(ctx context.Context, notificationID string) (dto.NotificationStatus, error)
	GetRecipientNotificationStatuses(ctx context.Context, filter providers.StatusFilters) ([]dto.RecipientNotificationStatus, error)
	GetNotificationTemplate(ctx context.Context, templateId string) (dto.NotificationTemplateDetails, error)
	GetDistributionListRecipients(ctx context.Context, name string) ([]string, error)
}

type NotificationInfoUpdater interface {
	UpdateNotificationStatus(ctx context.Context, log dto.NotificationStatusLog) error
	UpdateRecipientNotificationStatus(ctx context.Context, notificationID string, batch []dto.RecipientNotificationStatus) error
}

type QueueConsumer interface {
	Ack(ctx context.Context, deleteTag string) error
}

type InAppSender interface {
	SendNotifications(ctx context.Context, batch []dto.UserNotificationReq) error
}

type EmailSender interface {
	SendNotifications(ctx context.Context, batch []dto.UserEmailNotificationReq) error
}

type NotificationContents struct {
	Title    string
	Contents string
	Topic    string
	Image    *string
	IsHTML   bool
}

type notificationChannelParams[T any] struct {
	Channel           dto.NotificationChannel
	Contentents       NotificationContents
	UsersInfo         []providers.UserInfo
	NotificationReqFn func(userInfo providers.UserInfo, c NotificationContents) T
	SendNotifications func(ctx context.Context, batch []T) error
}

type WorkerCfg struct {
	UserInfoProvider         UserInfoProvider
	NotificationInfoProvider NotificationInfoProvider
	NotificationInfoUpdater  NotificationInfoUpdater
	Queue                    QueueConsumer
	InAppSender              InAppSender
	EmailSender              EmailSender
	NotificationChan         <-chan dto.NotificationMsg
}

type Worker struct {
	userInfoProvider         UserInfoProvider
	notificationInfoProvider NotificationInfoProvider
	notificationInfoUpdater  NotificationInfoUpdater
	queue                    QueueConsumer
	inAppSender              InAppSender
	emailSender              EmailSender
	notificationChan         <-chan dto.NotificationMsg
}

func NewWorker(cfg WorkerCfg) *Worker {
	return &Worker{
		userInfoProvider:         cfg.UserInfoProvider,
		notificationInfoProvider: cfg.NotificationInfoProvider,
		notificationInfoUpdater:  cfg.NotificationInfoUpdater,
		queue:                    cfg.Queue,
		inAppSender:              cfg.InAppSender,
		emailSender:              cfg.EmailSender,
		notificationChan:         cfg.NotificationChan,
	}
}

func (w *Worker) buildNotification(p dto.NotificationMsgPayload, t *dto.NotificationTemplateDetails) NotificationContents {

	contents := NotificationContents{
		Topic: p.Topic,
		Image: p.Image,
	}

	if p.RawContents != nil {
		contents.Title = p.RawContents.Title
		contents.Contents = p.RawContents.Contents

		return contents
	}

	contents.Title = t.TitleTemplate
	contents.Contents = t.ContentsTemplate
	contents.IsHTML = t.IsHtml

	for _, value := range p.TemplateContents.Variables {
		name := fmt.Sprintf("{{%s}}", value.Name)
		contents.Title = strings.ReplaceAll(contents.Title, name, value.Value)
		contents.Contents = strings.ReplaceAll(contents.Contents, name, value.Value)
	}

	return contents
}

func (w *Worker) failProcess(ctx context.Context, err error, notificationId string) {

	errArr := []error{err}

	notificatioStatus := dto.NotificationStatusLog{
		NotificationId: notificationId,
		Status:         dto.Failed,
	}

	err = w.notificationInfoUpdater.UpdateNotificationStatus(ctx, notificatioStatus)

	if err != nil {
		errArr = append(errArr, fmt.Errorf("failed to update notification status - %w", err))
	}

	slog.Error(errors.Join(errArr...).Error())
}

func (w *Worker) getRecipientsToSendNotifications(ctx context.Context, msg dto.NotificationMsg) ([]string, error) {

	recipients := make([]string, len(msg.Payload.Recipients))
	copy(recipients, msg.Payload.Recipients)

	if msg.Payload.DistributionList != nil {
		dlRecipients, err := w.notificationInfoProvider.GetDistributionListRecipients(
			ctx, *msg.Payload.DistributionList)

		if err != nil {
			slog.Error(err.Error())
			return nil, fmt.Errorf("failed to get distribution list recipients - %w", err)
		}

		recipients = append(recipients, dlRecipients...)
	}

	sentNotifications, err := w.notificationInfoProvider.GetRecipientNotificationStatuses(ctx, providers.StatusFilters{
		NotificationId: msg.Payload.Id,
		Channels:       msg.Payload.Channels,
		Statuses: []dto.NotificationStatus{
			dto.Sent,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get sent notifications - %w", err)
	}

	sentRecipients := map[string]struct{}{}

	for _, sentNotification := range sentNotifications {
		sentRecipients[sentNotification.UserId] = struct{}{}
	}

	recipientsToSend := []string{}

	for _, recipient := range recipients {
		if _, ok := sentRecipients[recipient]; !ok {
			recipientsToSend = append(recipientsToSend, recipient)
		}
	}

	return recipientsToSend, nil
}

func processChannelNotifications[T any](ctx context.Context, params notificationChannelParams[T]) ([]dto.RecipientNotificationStatus, bool) {
	notifications := make([]T, 0, len(params.UsersInfo))
	recipientStatusLogs := make([]dto.RecipientNotificationStatus, 0, len(params.UsersInfo))

	for _, userInfo := range params.UsersInfo {
		notification := params.NotificationReqFn(userInfo, params.Contentents)

		recipientStatusLogs = append(recipientStatusLogs, dto.RecipientNotificationStatus{
			UserId:  userInfo.UserId,
			Status:  string(dto.Sent),
			Channel: string(params.Channel),
		})

		notifications = append(notifications, notification)
	}

	err := params.SendNotifications(ctx, notifications)

	if err == nil {
		return recipientStatusLogs, false
	}

	slog.Error(fmt.Sprintf("failed to send %s notifications - %s",
		string(params.Channel), err.Error()))

	for i := range recipientStatusLogs {
		errMsg := fmt.Sprintf("failed to send in-app notification - %s", err.Error())
		recipientStatusLogs[i].Status = string(dto.Failed)
		recipientStatusLogs[i].ErrMsg = &errMsg
	}

	return recipientStatusLogs, true
}

func (w *Worker) processInAppNotification(ctx context.Context, usersInfo []providers.UserInfo, notification NotificationContents) ([]dto.RecipientNotificationStatus, bool) {

	makeInAppNotification := func(userInfo providers.UserInfo, c NotificationContents) dto.UserNotificationReq {
		return dto.UserNotificationReq{
			UserId:   userInfo.UserId,
			Title:    c.Title,
			Contents: c.Contents,
			Topic:    c.Topic,
			Image:    c.Image,
		}
	}

	params := notificationChannelParams[dto.UserNotificationReq]{
		Channel:           dto.InApp,
		Contentents:       notification,
		UsersInfo:         usersInfo,
		NotificationReqFn: makeInAppNotification,
		SendNotifications: w.inAppSender.SendNotifications,
	}

	recipientStatusLogs, hasFailed := processChannelNotifications(ctx, params)

	return recipientStatusLogs, hasFailed
}

func (w *Worker) processEmailNotification(ctx context.Context, usersInfo []providers.UserInfo, notification NotificationContents) ([]dto.RecipientNotificationStatus, bool) {

	makeEmailNotification := func(userInfo providers.UserInfo, c NotificationContents) dto.UserEmailNotificationReq {
		return dto.UserEmailNotificationReq{
			Email:  userInfo.Email,
			IsHtml: c.IsHTML,
			UserNotificationReq: dto.UserNotificationReq{
				UserId:   userInfo.UserId,
				Title:    c.Title,
				Contents: c.Contents,
				Topic:    c.Topic,
				Image:    c.Image,
			},
		}
	}

	params := notificationChannelParams[dto.UserEmailNotificationReq]{
		Channel:           dto.Email,
		Contentents:       notification,
		UsersInfo:         usersInfo,
		NotificationReqFn: makeEmailNotification,
		SendNotifications: w.emailSender.SendNotifications,
	}

	recipientStatusLogs, hasFailed := processChannelNotifications(ctx, params)

	return recipientStatusLogs, hasFailed
}

func (w *Worker) ProcessNotification(ctx context.Context, msg dto.NotificationMsg) {

	notificationId := msg.Payload.Id

	status, err := w.notificationInfoProvider.GetNotificationStatus(ctx, notificationId)

	if err != nil {
		err = fmt.Errorf("failed to get notification status - %w", err)
		w.failProcess(ctx, err, notificationId)
		return
	}

	if status == dto.Canceled {
		slog.Info("Notification is canceled, skipping")
		if err := w.queue.Ack(ctx, msg.DeleteTag); err != nil {
			err = fmt.Errorf("failed to ack message - %w", err)
			w.failProcess(ctx, err, notificationId)
			return
		}
		return
	}

	notificatioStatus := dto.NotificationStatusLog{
		NotificationId: notificationId,
		Status:         dto.Sending,
	}

	err = w.notificationInfoUpdater.UpdateNotificationStatus(ctx, notificatioStatus)

	if err != nil {
		err = fmt.Errorf("failed to update notification status - %w", err)
		w.failProcess(ctx, err, notificationId)
		return
	}

	recipients, err := w.getRecipientsToSendNotifications(ctx, msg)

	if err != nil {
		err = fmt.Errorf("failed to get recipients to send notifications - %w", err)
		w.failProcess(ctx, err, notificationId)
		return
	}

	if len(recipients) == 0 {
		slog.Info("No recipients to send notification, skipping")
		notificatioStatus.Status = dto.Sent

		if err := w.notificationInfoUpdater.UpdateNotificationStatus(ctx, notificatioStatus); err != nil {
			err = fmt.Errorf("failed to update notification status - %w", err)
			w.failProcess(ctx, err, notificationId)
			return
		}

		if err := w.queue.Ack(ctx, msg.DeleteTag); err != nil {
			err = fmt.Errorf("failed to ack message - %w", err)
			w.failProcess(ctx, err, notificationId)
			return
		}

		return
	}

	hasFailed := false
	userInfo := make([]providers.UserInfo, 0, len(recipients))

	for _, recipient := range recipients {
		info, err := w.userInfoProvider.GetUserInfo(ctx, recipient)

		if err != nil {
			hasFailed = true
			err = fmt.Errorf("failed to get user info - %w", err)
			slog.Error(err.Error())
			continue
		}

		userInfo = append(userInfo, info)
	}

	var templateDetails *dto.NotificationTemplateDetails

	if msg.Payload.TemplateContents != nil {
		details, err := w.notificationInfoProvider.GetNotificationTemplate(ctx, msg.Payload.TemplateContents.Id)

		if err != nil {
			err = fmt.Errorf("failed to get notification template - %w", err)
			w.failProcess(ctx, err, notificationId)
			return
		}

		templateDetails = &details
	}

	notification := w.buildNotification(msg.Payload, templateDetails)

	recipientStatusLogs := []dto.RecipientNotificationStatus{}

	notificatioStatus.Status = dto.Sent
	inAppStatusLogs, inAppHasFailed := w.processInAppNotification(ctx, userInfo, notification)
	recipientStatusLogs = append(recipientStatusLogs, inAppStatusLogs...)

	if inAppHasFailed {
		hasFailed = true
		notificatioStatus.Status = dto.Failed
	}

	emailStatusLogs, hasFailed := w.processEmailNotification(ctx, userInfo, notification)
	recipientStatusLogs = append(recipientStatusLogs, emailStatusLogs...)

	if inAppHasFailed {
		hasFailed = true
		notificatioStatus.Status = dto.Failed
	}

	if err := w.notificationInfoUpdater.UpdateNotificationStatus(ctx, notificatioStatus); err != nil {
		hasFailed = true
		err = fmt.Errorf("failed to update notification status - %w", err)
		w.failProcess(ctx, err, notificationId)
		return
	}

	if err := w.notificationInfoUpdater.UpdateRecipientNotificationStatus(ctx, notificationId, recipientStatusLogs); err != nil {
		hasFailed = true
		err = fmt.Errorf("failed to update recipient notification status - %w", err)
		w.failProcess(ctx, err, notificationId)
		return
	}

	if hasFailed {
		return
	}

	if err := w.queue.Ack(ctx, msg.DeleteTag); err != nil {
		err = fmt.Errorf("failed to ack message - %w", err)
		w.failProcess(ctx, err, notificationId)
		return
	}
}

func (w *Worker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case notification := <-w.notificationChan:
			w.ProcessNotification(ctx, notification)
		}
	}
}
