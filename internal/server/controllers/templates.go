package controllers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"

	"github.com/microcosm-cc/bluemonday"

	"github.com/notifique/internal/server/dto"
)

type NotificationTemplateRegistry interface {
	SaveTemplate(ctx context.Context, createdBy string, ntr dto.NotificationTemplateReq) (dto.NotificationTemplateCreatedResp, error)
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
