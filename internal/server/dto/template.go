package dto

type TemplateVariable struct {
	Name       string  `json:"name" binding:"required,max=120"`
	Type       string  `json:"type" binding:"required,oneof=STRING DATE TIME NUMBER"`
	Required   bool    `json:"required"`
	Validation *string `json:"validation"`
}

type NotificationTemplateReq struct {
	Name             string             `json:"name" binding:"required,max=120"`
	TitleTemplate    string             `json:"titleTemplate" binding:"required,max=120"`
	ContentsTemplate string             `json:"contentsTemplate" binding:"required,max=4096"`
	Description      string             `json:"description" binding:"required,max=256"`
	Variables        []TemplateVariable `json:"variables" binding:"unique_var_name,dive"`
}

type NotificationTemplateCreatedResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type NotificationTemplateFilters struct {
	PageFilter
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

type NotificationTemplateDetails struct {
	Id               string             `json:"id"`
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	TitleTemplate    string             `json:"titleTemplate"`
	ContentsTemplate string             `json:"contentsTemplate"`
	CreatedAt        string             `json:"createdAt"`
	CreatedBy        string             `json:"createdBy"`
	UpdatedAt        *string            `json:"updatedAt"`
	UpdatedBy        *string            `json:"updatedBy"`
	Variables        []TemplateVariable `json:"variables"`
}
