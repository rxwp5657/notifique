package controllers

import (
	"errors"
	"net/http"

	"golang.org/x/net/context"

	"github.com/gin-gonic/gin"
	"github.com/notifique/dto"
	"github.com/notifique/internal"
)

type DistributionListStorage interface {
	CreateDistributionList(ctx context.Context, distributionList dto.DistributionList) error
	GetDistributionLists(ctx context.Context, filter dto.PageFilter) ([]dto.DistributionListSummary, error)
	GetRecipients(ctx context.Context, distlistName string, filter dto.PageFilter) ([]string, error)
	AddRecipients(ctx context.Context, distlistName string, recipients []string) error
	DeleteRecipients(ctx context.Context, distlistName string, recipients []string) error
}

type DistributionListController struct {
	Storage DistributionListStorage
}

type recipientsHandler func(context.Context, string, []string) error

func (dc DistributionListController) CreateDistributionList(c *gin.Context) {
	var dl dto.DistributionList

	if err := c.ShouldBindJSON(&dl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dc.Storage.CreateDistributionList(c, dl); err != nil {
		if errors.As(err, &internal.DistributionListAlreadyExists{}) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			c.Status(http.StatusInternalServerError)
		}
	}

	c.Status(http.StatusCreated)
}

func (dc DistributionListController) GetDistributionLists(c *gin.Context) {
	var filters dto.PageFilter

	if err := c.ShouldBind(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lists, err := dc.Storage.GetDistributionLists(c, filters)

	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, lists)
}

func (dc DistributionListController) GetRecipients(c *gin.Context) {
	var uriParams dto.DistributionListUriParams

	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var filter dto.PageFilter

	if err := c.ShouldBind(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipients, err := dc.Storage.GetRecipients(c, uriParams.Name, filter)

	if err != nil {
		if errors.As(err, &internal.DistributionListNotFound{}) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		} else {
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	c.JSON(http.StatusOK, recipients)
}

func (dc DistributionListController) AddRecipients(c *gin.Context) {
	dc.handleRecipients(c, dc.Storage.AddRecipients)
}

func (dc DistributionListController) DeleteRecipients(c *gin.Context) {
	dc.handleRecipients(c, dc.Storage.DeleteRecipients)
}

func (dc DistributionListController) handleRecipients(c *gin.Context, handler recipientsHandler) {
	var uriParams dto.DistributionListUriParams

	if err := c.ShouldBindUri(&uriParams); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var recipients dto.DistributionListRecipients

	if err := c.ShouldBindJSON(&recipients); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := handler(c, uriParams.Name, recipients.Recipients)

	if err != nil {
		if errors.As(err, &internal.DistributionListNotFound{}) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	c.Status(http.StatusNoContent)
}
