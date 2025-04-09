package dto

type TemplateVariable struct {
	Name       string  `json:"name" binding:"required,max=120,templatevarname"`
	Type       string  `json:"type" binding:"required,oneof=STRING DATE DATETIME NUMBER"`
	Required   bool    `json:"required"`
	Validation *string `json:"validation"`
}

type NotificationTemplateDetails struct {
	Id               string             `json:"id"`
	Name             string             `json:"name"`
	IsHtml           bool               `json:"isHtml"`
	Description      string             `json:"description"`
	TitleTemplate    string             `json:"titleTemplate"`
	ContentsTemplate string             `json:"contentsTemplate"`
	CreatedAt        string             `json:"createdAt"`
	CreatedBy        string             `json:"createdBy"`
	UpdatedAt        *string            `json:"updatedAt"`
	UpdatedBy        *string            `json:"updatedBy"`
	Variables        []TemplateVariable `json:"variables"`
}
