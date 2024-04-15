package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/notifique/dto"
	"github.com/notifique/internal"
)

type NotificationStorage interface {
	SaveNotification(ctx context.Context, notification dto.NotificationReq) (string, error)
	GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) ([]dto.UserNotification, error)
	GetUserConfig(ctx context.Context, userId string) ([]dto.ChannelConfig, error)
	SetReadStatus(ctx context.Context, userId, notificationId string) error
	UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error
}

type NotificationController struct {
	Storage NotificationStorage
}

const userIdHeaderKey = "userId"

func (nc NotificationController) GetUserNotifications(c *gin.Context) {
	var filters dto.UserNotificationFilters

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filters.UserId = c.GetHeader(userIdHeaderKey)
	notifications, err := nc.Storage.GetUserNotifications(c, filters)

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func (nc NotificationController) GetUserConfig(c *gin.Context) {
	userId := c.GetHeader(userIdHeaderKey)
	cfg, err := nc.Storage.GetUserConfig(c, userId)

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, cfg)
}

func (nc NotificationController) CreateNotification(c *gin.Context) {
	var notification dto.NotificationReq

	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := nc.Storage.SaveNotification(c, notification); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func (nc NotificationController) SetReadStatus(c *gin.Context) {
	var n dto.NotificationUriParams

	if err := c.ShouldBindUri(&n); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := c.GetHeader(userIdHeaderKey)
	err := nc.Storage.SetReadStatus(c, userId, n.NotificationId)

	if err != nil {
		if errors.As(err, &internal.NotificationNotFound{}) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		} else {
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	c.Status(http.StatusOK)
}

func (nc NotificationController) UpdateUserConfig(c *gin.Context) {

	var userConfig dto.UserConfig

	if err := c.ShouldBindJSON(&userConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
