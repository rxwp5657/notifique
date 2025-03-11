package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"golang.org/x/net/context"

	"github.com/gin-gonic/gin"
	"github.com/notifique/internal"
	"github.com/notifique/internal/dto"
)

type UserRegistry interface {
	GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error)
	GetUserConfig(ctx context.Context, userId string) (dto.UserConfig, error)
	SetReadStatus(ctx context.Context, userId, notificationId string) error
	UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error
}

type UserNotificationBroker interface {
	Suscribe(ctx context.Context, userId string) (<-chan dto.UserNotification, error)
	Unsubscribe(ctx context.Context, userId string) error
	Publish(ctx context.Context, userId string, un dto.UserNotification) error
}

type UserController struct {
	Registry UserRegistry
	Broker   UserNotificationBroker
}

func (nc *UserController) GetUserNotifications(c *gin.Context) {
	var filters dto.UserNotificationFilters

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filters.UserId = c.GetHeader(UserIdHeaderKey)
	notifications, err := nc.Registry.GetUserNotifications(c, filters)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func (nc *UserController) GetUserConfig(c *gin.Context) {
	userId := c.GetHeader(UserIdHeaderKey)
	cfg, err := nc.Registry.GetUserConfig(c, userId)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, cfg)
}

func (nc *UserController) SetReadStatus(c *gin.Context) {
	var n dto.NotificationUriParams

	if err := c.ShouldBindUri(&n); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := c.GetHeader(UserIdHeaderKey)
	err := nc.Registry.SetReadStatus(c, userId, n.NotificationId)

	if err != nil {
		if errors.As(err, &internal.EntityNotFound{}) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		} else {
			slog.Error(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	c.Status(http.StatusOK)
}

func (nc *UserController) UpdateUserConfig(c *gin.Context) {

	var userConfig dto.UserConfig

	if err := c.ShouldBindJSON(&userConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := c.GetHeader(UserIdHeaderKey)

	if err := nc.Registry.UpdateUserConfig(c, userId, userConfig); err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

func (nc *UserController) GetLiveUserNotifications(c *gin.Context) {

	userId := c.GetHeader(UserIdHeaderKey)
	ch, err := nc.Broker.Suscribe(c, userId)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	slog.Info(fmt.Sprintf("user %s connected", userId))
	defer nc.Broker.Unsubscribe(c, userId)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		for {
			select {
			case <-c.Request.Context().Done():
				slog.Info(fmt.Sprintf("user %s disconnected", userId))
				return false
			case un := <-ch:
				marshalled, err := json.Marshal(un)
				if err != nil {
					slog.Error(err.Error())
					continue
				}
				c.SSEvent(userNotificationEvent, string(marshalled))
				return true
			}
		}
	})
}
