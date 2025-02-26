package controllers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"

	"github.com/microcosm-cc/bluemonday"

	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/dto"
)

type NotificationTemplateRegistry interface {
	SaveTemplate(ctx context.Context, createdBy string, ntr dto.NotificationTemplateReq) (dto.NotificationTemplateCreatedResp, error)
	GetTemplates(ctx context.Context, filters dto.NotificationTemplateFilters) (dto.Page[dto.NotificationTemplateInfoResp], error)
	GetTemplateDetails(ctx context.Context, id string) (dto.NotificationTemplateDetails, error)
	DeleteTemplate(ctx context.Context, id string) error
}

type NotificationTemplateController struct {
	Registry NotificationTemplateRegistry
}

type Sanitizer func(s string) string

var sanitize Sanitizer = func() Sanitizer {
	p := bluemonday.UGCPolicy()

	return func(s string) string {
		sanitized := p.Sanitize(s)

		if sanitized != "" {
			return sanitized
		}

		return s
	}
}()

func (ntc *NotificationTemplateController) CreateNotificationTemplate(c *gin.Context) {

	var ntr dto.NotificationTemplateReq

	if err := c.ShouldBindJSON(&ntr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ntr.ContentsTemplate = sanitize(ntr.ContentsTemplate)
	ntr.TitleTemplate = sanitize(ntr.TitleTemplate)

	userId := c.GetHeader(UserIdHeaderKey)

	resp, err := ntc.Registry.SaveTemplate(c.Request.Context(), userId, ntr)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (ntc *NotificationTemplateController) GetTemplates(c *gin.Context) {
	var filters dto.NotificationTemplateFilters

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notifications, err := ntc.Registry.GetTemplates(c.Request.Context(), filters)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
	}

	c.JSON(http.StatusOK, notifications)
}

func (ntc *NotificationTemplateController) GetTemplateDetails(c *gin.Context) {
	var params dto.NotificationTemplateUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification, err := ntc.Registry.GetTemplateDetails(c.Request.Context(), params.Id)

	if err != nil {
		if errors.As(err, &server.EntityNotFound{}) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		} else {
			slog.Error(err.Error())
			c.Status(http.StatusInternalServerError)
		}
	}

	c.JSON(http.StatusOK, notification)
}

func (ntc *NotificationTemplateController) DeleteTemplate(c *gin.Context) {

	var params dto.NotificationTemplateUriParams

	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := ntc.Registry.DeleteTemplate(c.Request.Context(), params.Id)

	if err != nil {
		slog.Error(err.Error())
		c.Status(http.StatusInternalServerError)
	}

	c.Status(http.StatusNoContent)
}
