package controllers

import (
	"context"
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

type NotificationRegistry interface {
	SaveNotification(ctx context.Context, createdBy string, notification sdto.NotificationReq) (string, error)
	UpdateNotificationStatus(ctx context.Context, statusLog sdto.NotificationStatusLog) error
	DeleteNotification(ctx context.Context, id string) error
	GetNotificationStatus(ctx context.Context, notificationId string) (sdto.NotificationStatus, error)
	GetTemplateVariables(ctx context.Context, templateId string) ([]sdto.TemplateVariable, error)
	GetNotifications(ctx context.Context, filters sdto.PageFilter) (sdto.Page[dto.NotificationSummary], error)
	GetNotification(ctx context.Context, notificationId string) (dto.NotificationResp, error)
	UpsertRecipientNotificationStatuses(ctx context.Context, notificationId string, statuses []sdto.RecipientNotificationStatus) error
	GetRecipientNotificationStatuses(ctx context.Context, notificationId string, filters sdto.NotificationRecipientStatusFilters) (sdto.Page[sdto.RecipientNotificationStatus], error)
}

type NotificationPublisher interface {
	Publish(ctx context.Context, notification sdto.NotificationMsgPayload) error
}

type NotificationController struct {
	Registry  NotificationRegistry
	Publisher NotificationPublisher
	Cache     cache.Cache
}

const SendingNotificationMsg = "Notification is being sent"
const SentNotificationMsg = "Notification has been sent"

func notificationExists(ctx context.Context, c cache.Cache, hash string) (bool, error) {
	_, err, ok := c.Get(ctx, cache.GetHashKey(hash))

	if err != nil {
		return false, err
	}

	return ok, nil
}

func getCachedNotificationStatus(ctx context.Context, c cache.Cache, notificationId string) (*sdto.NotificationStatus, error) {
	status, err, ok := c.Get(ctx, cache.GetNotificationStatusKey(notificationId))

	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}

	s := sdto.NotificationStatus(status)
	return &s, nil
}

func setNotificationHash(ctx context.Context, c cache.Cache, hash string) error {
	key := cache.GetHashKey(hash)
	return c.Set(ctx, key, "1", NotificationHashTTL)
}

func deleteNotificationHash(ctx context.Context, c cache.Cache, hash string) error {
	key := cache.GetHashKey(hash)
	return c.Del(ctx, key)
}

func UpdateNotificationStatus(ctx context.Context, c cache.Cache, sl sdto.NotificationStatusLog) error {
	key := cache.GetNotificationStatusKey(sl.NotificationId)
	return c.Set(ctx, key, string(sl.Status), NotificationStatusTTL)
}

func (nc *NotificationController) getNotificationStatus(ctx context.Context, notificationId string) (sdto.NotificationStatus, error) {

	status, err := getCachedNotificationStatus(ctx, nc.Cache, notificationId)

	if err != nil {
		return sdto.NotificationStatus(""), fmt.Errorf("failed to get cached notification status: %w", err)
	}

	if status == nil {
		dbStatus, err := nc.Registry.GetNotificationStatus(ctx, notificationId)

		if err != nil && errors.As(err, &internal.EntityNotFound{}) {
			return sdto.NotificationStatus(""), err
		} else if err != nil {
			return sdto.NotificationStatus(""), fmt.Errorf("failed to get notification status from registry: %w", err)
		}

		status = &dbStatus
	}

	return *status, nil
}

func (nc *NotificationController) CreateNotification(c *gin.Context) {
	var notificationReq sdto.NotificationReq

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

	hash := internal.GetMd5Hash(string(body))

	if exists, err := notificationExists(c.Request.Context(), nc.Cache, hash); err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	} else if exists {
		c.Status(http.StatusNoContent)
		return
	}

	userId := c.GetHeader(string(auth.UserHeader))

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

	statusLog := sdto.NotificationStatusLog{
		NotificationId: notificationId,
		Status:         sdto.Created,
	}

	if err := UpdateNotificationStatus(context.TODO(), nc.Cache, statusLog); err != nil {
		slog.Error(err.Error())
	}

	if err := setNotificationHash(c.Request.Context(), nc.Cache, hash); err != nil {
		slog.Error("Failed to set notification hash",
			"error", err.Error(),
			"notificationId", notificationId)
	}

	go func() {
		ctx := context.Background()

		notificationWithId := sdto.NotificationMsgPayload{
			Id:              notificationId,
			Hash:            hash,
			NotificationReq: notificationReq,
		}

		if err := nc.Publisher.Publish(ctx, notificationWithId); err != nil {
			slog.Error("Failed to publish notification",
				"error", err.Error(),
				"notificationId", notificationId)

			if err := deleteNotificationHash(ctx, nc.Cache, hash); err != nil {
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

	invalidStatuses := map[sdto.NotificationStatus]string{
		sdto.Sending: SendingNotificationMsg,
		sdto.Sent:    SentNotificationMsg,
	}

	status, err := nc.getNotificationStatus(c.Request.Context(), params.NotificationId)

	if err != nil && errors.As(err, &internal.EntityNotFound{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	} else if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	if msg, ok := invalidStatuses[status]; ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	statusLog := sdto.NotificationStatusLog{
		NotificationId: params.NotificationId,
		Status:         sdto.Canceled,
	}

	// Update both cache and registry
	if err := UpdateNotificationStatus(context.TODO(), nc.Cache, statusLog); err != nil {
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
	var filters sdto.PageFilter

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

func (nc *NotificationController) UpsertRecipientNotificationStatuses(c *gin.Context) {
	var params dto.NotificationUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var statuses []sdto.RecipientNotificationStatus

	if err := c.ShouldBindJSON(&statuses); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := nc.Registry.UpsertRecipientNotificationStatuses(
		c.Request.Context(),
		params.NotificationId,
		statuses)

	if err != nil && errors.As(err, &internal.EntityNotFound{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	} else if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func (nc *NotificationController) GetRecipientNotificationStatuses(c *gin.Context) {
	var params dto.NotificationUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var filters sdto.NotificationRecipientStatusFilters

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logs, err := nc.Registry.GetRecipientNotificationStatuses(c.Request.Context(), params.NotificationId, filters)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (nc *NotificationController) UpdateStatus(c *gin.Context) {

	var params dto.NotificationUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var statusLog sdto.NotificationStatusLog

	if err := c.ShouldBindJSON(&statusLog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := nc.Registry.UpdateNotificationStatus(c.Request.Context(), statusLog)

	if err != nil && errors.As(err, &internal.EntityNotFound{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	} else if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := UpdateNotificationStatus(c.Request.Context(), nc.Cache, statusLog); err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

func (nc *NotificationController) GetStatus(c *gin.Context) {

	var params dto.NotificationUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := nc.getNotificationStatus(c.Request.Context(), params.NotificationId)

	if err != nil && errors.As(err, &internal.EntityNotFound{}) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	} else if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, sdto.NotificationStatusResp{
		Status: status,
	})
}
