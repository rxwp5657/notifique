package dto

import (
	sdto "github.com/notifique/shared/dto"
)

type TemplateVariableType string

const (
	String   TemplateVariableType = "STRING"
	Date     TemplateVariableType = "DATE"
	DateTime TemplateVariableType = "DATETIME"
	Number   TemplateVariableType = "NUMBER"
)

type NotificationTemplateReq struct {
	Name             string                  `json:"name" binding:"required,max=120"`
	IsHtml           bool                    `json:"isHtml" binding:"required"`
	TitleTemplate    string                  `json:"titleTemplate" binding:"required,max=120"`
	ContentsTemplate string                  `json:"contentsTemplate" binding:"required,max=4096"`
	Description      string                  `json:"description" binding:"required,max=256"`
	Variables        []sdto.TemplateVariable `json:"variables" binding:"unique_var_name,dive"`
}

type NotificationTemplateCreatedResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type NotificationTemplateFilters struct {
	sdto.PageFilter
	TemplateName *string `form:"templateName" binding:"omitempty"`
}

type NotificationTemplateInfoResp struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type NotificationTemplateUriParams struct {
	Id string `uri:"id" binding:"uuid"`
}
