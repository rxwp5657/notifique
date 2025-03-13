package controllers

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/notifique/internal"
	"github.com/notifique/internal/dto"
)

type NotificationMsg struct {
	dto.NotificationReq
	Id   string `json:"id"`
	Hash string `json:"hash"`
}

type NotificationStatusLog struct {
	NotificationId string
	Status         dto.NotificationStatus
	ErrorMsg       *string
}

type NotificationRegistry interface {
	SaveNotification(ctx context.Context, createdBy string, notification dto.NotificationReq) (string, error)
	UpdateNotificationStatus(ctx context.Context, statusLog NotificationStatusLog) error
	DeleteNotification(ctx context.Context, id string) error
	GetNotificationStatus(ctx context.Context, notificationId string) (dto.NotificationStatus, error)
	GetTemplateVariables(ctx context.Context, templateId string) ([]dto.TemplateVariable, error)
	GetNotifications(ctx context.Context, filters dto.PageFilter) (dto.Page[dto.NotificationSummary], error)
	GetNotification(ctx context.Context, notificationId string) (dto.NotificationResp, error)
}

type NotificationPublisher interface {
	Publish(ctx context.Context, notification NotificationMsg) error
}

type NotificationCache interface {
	GetNotificationStatus(ctx context.Context, notificationId string) (*dto.NotificationStatus, error)
	UpdateNotificationStatus(ctx context.Context, statusLog NotificationStatusLog) error
	NotificationExists(ctx context.Context, hash string) (bool, error)
	SetNotificationHash(ctx context.Context, hash string) error
	DeleteNotificationHash(ctx context.Context, hash string) error
}

type NotificationController struct {
	Registry  NotificationRegistry
	Publisher NotificationPublisher
	Cache     NotificationCache
}

const SendingNotificationMsg = "Notification is being sent"
const SentNotificationMsg = "Notification has been sent"

func makeNotificationHash(body []byte) string {
	hash := md5.Sum(body)
	return hex.EncodeToString(hash[:])
}

func (nc *NotificationController) CreateNotification(c *gin.Context) {
	var notificationReq dto.NotificationReq

	if err := c.ShouldBindJSON(&notificationReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	body, err := io.ReadAll(c.Request.Body)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	hash := makeNotificationHash(body)

	if exists, err := nc.Cache.NotificationExists(c.Request.Context(), hash); err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	} else if exists {
		c.Status(http.StatusNoContent)
		return
	}

	userId := c.GetHeader(UserIdHeaderKey)

	if notificationReq.TemplateContents != nil {
		templateId := notificationReq.TemplateContents.Id

		templateVars, err := nc.Registry.GetTemplateVariables(
			c.Request.Context(),
			templateId,
		)

		if err != nil && errors.As(err, &internal.EntityNotFound{}) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else if err != nil {
			slog.Error(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}

		suppliedVars := notificationReq.TemplateContents.Variables
		err = internal.ValidateTemplateVars(templateVars, suppliedVars)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	notificationId, err := nc.Registry.SaveNotification(c, userId, notificationReq)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)

	statusLog := NotificationStatusLog{
		NotificationId: notificationId,
		Status:         dto.Created,
	}

	if err := nc.Cache.UpdateNotificationStatus(context.TODO(), statusLog); err != nil {
		slog.Error(err.Error())
	}

	if err := nc.Cache.SetNotificationHash(c.Request.Context(), hash); err != nil {
		slog.Error("Failed to set notification hash",
			"error", err.Error(),
			"notificationId", notificationId)
	}

	go func() {
		ctx := context.Background()

		notificationWithId := NotificationMsg{
			NotificationReq: notificationReq,
			Id:              notificationId,
		}

		if err := nc.Publisher.Publish(ctx, notificationWithId); err != nil {
			slog.Error("Failed to publish notification",
				"error", err.Error(),
				"notificationId", notificationId)

			if err := nc.Cache.DeleteNotificationHash(ctx, hash); err != nil {
				slog.Error("Failed to delete notification hash",
					"error", err.Error(),
					"notificationId", notificationId)
			}
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
		return
	}

	c.Status(http.StatusNoContent)
}

func (nc *NotificationController) CancelDelivery(c *gin.Context) {
	var params dto.NotificationUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invalidStatuses := map[dto.NotificationStatus]string{
		dto.Sending: SendingNotificationMsg,
		dto.Sent:    SentNotificationMsg,
	}

	status, err := nc.Cache.GetNotificationStatus(c.Request.Context(), params.NotificationId)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	if status == nil {
		dbStatus, err := nc.Registry.GetNotificationStatus(c.Request.Context(), params.NotificationId)

		if err != nil && errors.As(err, &internal.EntityNotFound{}) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		} else if err != nil {
			slog.Error(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}

		status = &dbStatus
	}

	if msg, ok := invalidStatuses[*status]; ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	statusLog := NotificationStatusLog{
		NotificationId: params.NotificationId,
		Status:         dto.Canceled,
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

func (nc *NotificationController) GetNotifications(c *gin.Context) {
	var filters dto.PageFilter

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notifications, err := nc.Registry.GetNotifications(c.Request.Context(), filters)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, notifications)
}

func (nc *NotificationController) GetNotification(c *gin.Context) {
	var params dto.NotificationUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification, err := nc.Registry.GetNotification(c.Request.Context(), params.NotificationId)

	if err != nil && errors.As(err, &internal.EntityNotFound{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	} else if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, notification)
}
