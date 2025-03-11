package dto

type NotificationChannel string
type NotificationPriority string
type NotificationStatus string
type NotificationContentsType string

const (
	Created  NotificationStatus = "CREATED"
	Queued   NotificationStatus = "QUEUED"
	Failed   NotificationStatus = "FAILED"
	Sending  NotificationStatus = "SENDING"
	Sent     NotificationStatus = "SENT"
	Canceled NotificationStatus = "CANCELED"

	Email NotificationChannel = "e-mail"
	InApp NotificationChannel = "in-app"

	High   NotificationPriority = "HIGH"
	Medium NotificationPriority = "MEDIUM"
	Low    NotificationPriority = "LOW"

	Template NotificationContentsType = "TEMPLATE"
	Raw      NotificationContentsType = "RAW"
)

type RawContents struct {
	Title    string `json:"title" binding:"required,max=120"`
	Contents string `json:"contents" binding:"required,max=1024"`
}

type TemplateVariableContents struct {
	Name  string `json:"name" binding:"required,max=120,templatevarname"`
	Value string `json:"value" binding:"required"`
}

type TemplateContents struct {
	Id        string                     `json:"id" binding:"required,uuid"`
	Variables []TemplateVariableContents `json:"variables" binding:"required,unique,dive"`
}

type NotificationReq struct {
	RawContents      *RawContents          `json:"contents" binding:"required_without=TemplateContents,excluded_with=TemplateContents"`
	TemplateContents *TemplateContents     `json:"template" binding:"required_without=RawContents,excluded_with=RawContents"`
	Image            *string               `json:"image" binding:"omitempty,uri"`
	Topic            string                `json:"topic" binding:"required,min=1,max=120"`
	Priority         NotificationPriority  `json:"priority" binding:"oneof=HIGH MEDIUM LOW"`
	DistributionList *string               `json:"distributionList" binding:"omitempty,max=120,min=3,distributionlistname"`
	Recipients       []string              `json:"recipients" binding:"unique,max=256,dive,min=1"`
	Channels         []NotificationChannel `json:"channels" binding:"unique,dive,oneof=e-mail sms in-app"`
}

type NotificationUriParams struct {
	NotificationId string `uri:"id" binding:"required,uuid"`
}

type NotificationSummary struct {
	Id           string                   `json:"id"`
	Topic        string                   `json:"topic"`
	ContentsType NotificationContentsType `json:"contentsType"`
	Priority     NotificationPriority     `json:"priority"`
	Status       NotificationStatus       `json:"status"`
	CreatedAt    string                   `json:"createdAt"`
	CreatedBy    string                   `json:"createdBy"`
}

type NotificationResp struct {
	NotificationReq
	Id        string             `json:"id"`
	Status    NotificationStatus `json:"status"`
	CreatedAt string             `json:"createdAt"`
	CreatedBy string             `json:"createdBy"`
}

func (c NotificationChannel) ToStrSlice(channels []NotificationChannel) []string {
	strChannels := make([]string, 0, len(channels))

	for _, channel := range channels {
		strChannels = append(strChannels, string(channel))
	}

	return strChannels
}

func (p NotificationPriority) ToStrSlice(priorities []NotificationPriority) []string {
	strPriorities := make([]string, 0, len(priorities))

	for _, priority := range priorities {
		strPriorities = append(strPriorities, string(priority))
	}

	return strPriorities
}

func (s NotificationStatus) ToStrSlice(statuses []NotificationStatus) []string {
	strStatuses := make([]string, 0, len(statuses))

	for _, status := range statuses {
		strStatuses = append(strStatuses, string(status))
	}

	return strStatuses
}
