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
	Created NotificationStatus = "CREATED"
	Queued  NotificationStatus = "QUEUED"
	Failed  NotificationStatus = "FAILED"
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
}

type NotificationPublisher interface {
	Publish(ctx context.Context, notification Notification) error
}

type NotificationController struct {
	Registry  NotificationRegistry
	Publisher NotificationPublisher
}

func (nc NotificationController) CreateNotification(c *gin.Context) {
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
