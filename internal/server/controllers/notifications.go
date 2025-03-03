package controllers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/notifique/internal/server/dto"
)

type NotificationStatus string

const (
	Created  NotificationStatus = "CREATED"
	Queued   NotificationStatus = "QUEUED"
	Failed   NotificationStatus = "FAILED"
	Sending  NotificationStatus = "SENDING"
	Sent     NotificationStatus = "SENT"
	Canceled NotificationStatus = "CANCELED"
)

type Notification struct {
	dto.NotificationReq
	Id string `json:"id"`
}

type NotificationStatusLog struct {
	NotificationId string
	Status         NotificationStatus
	ErrorMsg       *string
}

type NotificationRegistry interface {
	SaveNotification(ctx context.Context, createdBy string, notification dto.NotificationReq) (string, error)
	UpdateNotificationStatus(ctx context.Context, statusLog NotificationStatusLog) error
	DeleteNotification(ctx context.Context, id string) error
	GetNotificationStatus(ctx context.Context, notificationId string) (*NotificationStatus, error)
}

type NotificationPublisher interface {
	Publish(ctx context.Context, notification Notification) error
}

type NotificationCache interface {
	GetNotificationStatus(ctx context.Context, notificationId string) (*NotificationStatus, error)
	UpdateNotificationStatus(ctx context.Context, statusLog NotificationStatusLog) error
}

type NotificationController struct {
	Registry  NotificationRegistry
	Publisher NotificationPublisher
	Cache     NotificationCache
}

const SendingNotificationMsg = "Notification is being sent"
const SentNotificationMsg = "Notification has been sent"

func (nc *NotificationController) CreateNotification(c *gin.Context) {
	var notification dto.NotificationReq

	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := c.GetHeader(UserIdHeaderKey)

	notificationId, err := nc.Registry.SaveNotification(c, userId, notification)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)

	statusLog := NotificationStatusLog{
		NotificationId: notificationId,
		Status:         Created,
	}

	if err := nc.Cache.UpdateNotificationStatus(context.TODO(), statusLog); err != nil {
		slog.Error(err.Error())
	}

	go func() {
		ctx := context.Background()

		notificationWithId := Notification{
			NotificationReq: notification,
			Id:              notificationId,
		}

		if err := nc.Publisher.Publish(ctx, notificationWithId); err != nil {
			slog.Error("Failed to publish notification",
				"error", err.Error(),
				"notificationId", notificationId)
		}
	}()
}

func (nc *NotificationController) DeleteNotification(c *gin.Context) {
	var params dto.NotificationUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := nc.Registry.DeleteNotification(c.Request.Context(), params.NotificationId)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
	}

	c.Status(http.StatusNoContent)
}

func (nc *NotificationController) CancelDelivery(c *gin.Context) {
	var params dto.NotificationUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invalidStatuses := map[NotificationStatus]string{
		Sending: SendingNotificationMsg,
		Sent:    SentNotificationMsg,
	}

	status, err := nc.Cache.GetNotificationStatus(c.Request.Context(), params.NotificationId)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	if status == nil {
		status, err = nc.Registry.GetNotificationStatus(c.Request.Context(), params.NotificationId)
		if err != nil {
			slog.Error(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	if msg, ok := invalidStatuses[*status]; ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	statusLog := NotificationStatusLog{
		NotificationId: params.NotificationId,
		Status:         Canceled,
	}

	// Update both cache and registry
	if err := nc.Cache.UpdateNotificationStatus(context.TODO(), statusLog); err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := nc.Registry.UpdateNotificationStatus(context.TODO(), statusLog); err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
