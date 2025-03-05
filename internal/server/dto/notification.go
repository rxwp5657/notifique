package dto

type RawContents struct {
	Title    string `json:"title" binding:"required,max=120"`
	Contents string `json:"contents" binding:"required,max=1024"`
}

type TemplateVariableContents struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
}

type TemplateContents struct {
	Id        string                     `json:"id" binding:"required,uuid"`
	Variables []TemplateVariableContents `json:"variables" binding:"required,unique,dive"`
}

type NotificationReq struct {
	RawContents      *RawContents      `json:"contents" binding:"required_without=TemplateContents,excluded_with=TemplateContents"`
	TemplateContents *TemplateContents `json:"template" binding:"required_without=RawContents,excluded_with=RawContents"`
	Image            *string           `json:"image" binding:"omitempty,uri"`
	Topic            string            `json:"topic" binding:"required,min=1,max=120"`
	Priority         string            `json:"priority" binding:"oneof=HIGH MEDIUM LOW"`
	DistributionList *string           `json:"distributionList" binding:"omitempty,max=120,min=3,distributionListName"`
	Recipients       []string          `json:"recipients" binding:"unique,max=256,dive,min=1"`
	Channels         []string          `json:"channels" binding:"unique,dive,oneof=e-mail sms in-app"`
}

type NotificationUriParams struct {
	NotificationId string `uri:"id" binding:"required,uuid"`
}
