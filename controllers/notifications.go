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
	GetNotifications(ctx context.Context, filters dto.NotificationFilters) ([]dto.UserNotificationResp, error)
	SetReadStatus(ctx context.Context, userId, notificationId string) error
	OptOut(ctx context.Context, userId string, channels []string) error
	OptIn(ctx context.Context, userId string, channels []string) error
}

type NotificationController struct {
	Storage NotificationStorage
}

func (nc NotificationController) GetNotifications(c *gin.Context) {
	var filters dto.NotificationFilters

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filters.UserId = "1231"
	notifications, err := nc.Storage.GetNotifications(c, filters)

	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func (nc NotificationController) CreateNotification(c *gin.Context) {
	var notification dto.NotificationReq

	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := nc.Storage.SaveNotification(c, notification); err != nil {
		c.JSON(http.StatusInternalServerError, nil)
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

	err := nc.Storage.SetReadStatus(c, "1231", n.NotificationId)

	if err != nil {
		if errors.As(err, &internal.NotificationNotFound{}) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		} else if errors.As(err, &internal.RecipientNotFound{}) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		} else {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
	}

	c.Status(http.StatusOK)
}

func (nc NotificationController) OptIn(c *gin.Context) {
	var ch dto.ChannelsReq

	if err := c.ShouldBindJSON(&ch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := nc.Storage.OptIn(c, "1231", ch.Channels); err != nil {
		c.JSON(http.StatusInternalServerError, nil)
	}

	c.Status(http.StatusOK)
}

func (nc NotificationController) OptOut(c *gin.Context) {
	var ch dto.ChannelsReq

	if err := c.ShouldBindJSON(&ch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := nc.Storage.OptOut(c, "1231", ch.Channels); err != nil {
		c.JSON(http.StatusInternalServerError, nil)
	}

	c.Status(http.StatusOK)
}
