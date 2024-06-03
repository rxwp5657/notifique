package controllers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/notifique/dto"
)

type NotificationStatus string

const (
	CREATED        NotificationStatus = "CREATED"
	PUBLISHED      NotificationStatus = "PUBLISHED"
	PUBLISH_FAILED NotificationStatus = "FAILED"
)

type Notification struct {
	dto.NotificationReq
	Id string `json:"id"`
}

type NotificationStorage interface {
	SaveNotification(ctx context.Context, createdBy string, notification dto.NotificationReq) (string, error)
	CreateNotificationStatusLog(ctx context.Context, notificationId string, status NotificationStatus, errMsg *string) error
}

type NotificationPublisher interface {
	Publish(ctx context.Context, notification Notification, storage NotificationStorage) error
}

type NotificationController struct {
	Storage   NotificationStorage
	Publisher NotificationPublisher
}

func (nc NotificationController) CreateNotification(c *gin.Context) {
	var notification dto.NotificationReq

	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := c.GetHeader(USER_ID_HEADER_KEY)

	notificationId, err := nc.Storage.SaveNotification(c, userId, notification)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	notificationWithId := Notification{
		NotificationReq: notification,
		Id:              notificationId,
	}

	err = nc.Publisher.Publish(c, notificationWithId, nc.Storage)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
