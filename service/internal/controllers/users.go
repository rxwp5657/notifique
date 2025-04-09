package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/shared/auth"
	"github.com/notifique/shared/cache"
	sdto "github.com/notifique/shared/dto"
)

type UserRegistry interface {
	GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (sdto.Page[dto.UserNotification], error)
	GetUserConfig(ctx context.Context, userId string) (dto.UserConfig, error)
	SetReadStatus(ctx context.Context, userId, notificationId string) error
	UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error
	CreateNotifications(ctx context.Context, notifications []sdto.UserNotificationReq) ([]dto.UserNotification, error)
}

type UserNotificationBroker interface {
	Suscribe(ctx context.Context, userId string) (<-chan dto.UserNotification, error)
	Unsubscribe(ctx context.Context, userId string) error
	Publish(ctx context.Context, userId string, un dto.UserNotification) error
}

type UserController struct {
	Registry UserRegistry
	Broker   UserNotificationBroker
	Cache    cache.Cache
}

func (nc *UserController) GetUserNotifications(c *gin.Context) {
	var filters dto.UserNotificationFilters

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filters.UserId = c.GetHeader(string(auth.UserHeader))
	notifications, err := nc.Registry.GetUserNotifications(c, filters)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func (nc *UserController) GetUserConfig(c *gin.Context) {
	userId := c.GetHeader(string(auth.UserHeader))
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

	userId := c.GetHeader(string(auth.UserHeader))
	err := nc.Registry.SetReadStatus(c, userId, n.NotificationId)

	if err != nil && errors.As(err, &internal.EntityNotFound{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	} else if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)

	path, _ := internal.GetBasePath(c.Request.URL.Path, ".*/users/me/notifications")

	// Delete cached user notifications
	err = nc.Cache.DelWithPrefix(
		c.Request.Context(),
		cache.GetEndpointKeyWithPrefix(path, &userId))

	if err != nil {
		err = fmt.Errorf("error deleting cached user notifications: %w", err)
		slog.Error(err.Error())
	}
}

func (nc *UserController) UpdateUserConfig(c *gin.Context) {

	var userConfig dto.UserConfig

	if err := c.ShouldBindJSON(&userConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := c.GetHeader(string(auth.UserHeader))

	if err := nc.Registry.UpdateUserConfig(c, userId, userConfig); err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)

	// Delete cached user notification config
	err := nc.Cache.DelWithPrefix(
		c.Request.Context(),
		cache.GetEndpointKeyWithPrefix(c.Request.URL.Path, &userId))

	if err != nil {
		err = fmt.Errorf("error deleting cached user notification config: %w", err)
		slog.Error(err.Error())
	}
}

func (nc *UserController) CreateNotifications(c *gin.Context) {

	batch := []sdto.UserNotificationReq{}

	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notifications, err := nc.Registry.CreateNotifications(c, batch)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)

	// Might be able to improve cache performance if we use a
	// data structure that stores all notifications sorted by
	// createdAt date. We would only have to push the new
	// notification
	for i, n := range batch {
		userId := n.UserId
		// Delete cached user notification config
		err := nc.Cache.DelWithPrefix(
			c.Request.Context(),
			cache.GetEndpointKeyWithPrefix("/users/me/notifications", &userId))

		if err != nil {
			err = fmt.Errorf("error deleting cached user notification config: %w", err)
			slog.Error(err.Error())
		}

		// Publish notification to user
		if err := nc.Broker.Publish(c, userId, notifications[i]); err != nil {
			slog.Error(err.Error())
			continue
		}
	}
}

func (nc *UserController) GetLiveUserNotifications(c *gin.Context) {

	userId := c.GetHeader(string(auth.UserHeader))
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
