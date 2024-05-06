package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"github.com/gin-gonic/gin"
	"github.com/notifique/dto"
	"github.com/notifique/internal"
)

type UserStorage interface {
	CreateUserNotification(ctx context.Context, userId string, notification dto.UserNotificationReq) (string, error)
	GetUserNotifications(ctx context.Context, filters dto.UserNotificationFilters) (dto.Page[dto.UserNotification], error)
	GetUserConfig(ctx context.Context, userId string) ([]dto.ChannelConfig, error)
	SetReadStatus(ctx context.Context, userId, notificationId string) error
	UpdateUserConfig(ctx context.Context, userId string, config dto.UserConfig) error
}

type UserController struct {
	Storage UserStorage
}

func (nc UserController) GetUserNotifications(c *gin.Context) {
	var filters dto.UserNotificationFilters

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filters.UserId = c.GetHeader(USER_ID_HEADER_KEY)
	notifications, err := nc.Storage.GetUserNotifications(c, filters)

	fmt.Println(err)

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func (nc UserController) CreateUserNotification(c *gin.Context) {
	var uriParam dto.UserNotificationUriParam

	if err := c.ShouldBindUri(&uriParam); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userNotification dto.UserNotificationReq

	if err := c.ShouldBindJSON(&userNotification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := nc.Storage.CreateUserNotification(c, uriParam.Id, userNotification)

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusCreated)
}

func (nc UserController) GetUserConfig(c *gin.Context) {
	userId := c.GetHeader(USER_ID_HEADER_KEY)
	cfg, err := nc.Storage.GetUserConfig(c, userId)

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, cfg)
}

func (nc UserController) SetReadStatus(c *gin.Context) {
	var n dto.NotificationUriParams

	if err := c.ShouldBindUri(&n); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := c.GetHeader(USER_ID_HEADER_KEY)
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

func (nc UserController) UpdateUserConfig(c *gin.Context) {

	var userConfig dto.UserConfig

	if err := c.ShouldBindJSON(&userConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
