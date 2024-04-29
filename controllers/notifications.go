package controllers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/notifique/dto"
)

type NotificationStorage interface {
	SaveNotification(ctx context.Context, createdBy string, notification dto.NotificationReq) (string, error)
}

type NotificationController struct {
	Storage NotificationStorage
}

func (nc NotificationController) CreateNotification(c *gin.Context) {
	var notification dto.NotificationReq

	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId := c.GetHeader(USER_ID_HEADER_KEY)
	if _, err := nc.Storage.SaveNotification(c, userId, notification); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
