package dto

type NotificationChannel string
type NotificationPriority string
type NotificationStatus string

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

type NotificationMsgPayload struct {
	NotificationReq
	Id   string `json:"id"`
	Hash string `json:"hash"`
}

type NotificationMsg struct {
	MessageId string
	DeleteTag string
	Payload   NotificationMsgPayload
}

type NotificationStatusLog struct {
	NotificationId string             `json:"notificationId" binding:"required,uuid"`
	Status         NotificationStatus `json:"status" binding:"required,oneof=CREATED QUEUED FAILED SENDING SENT CANCELED"`
	ErrorMsg       *string            `json:"errorMsg" binding:"omitempty"`
}

type RecipientNotificationStatus struct {
	UserId  string  `json:"userId" binding:"required"`
	Channel string  `json:"channel" binding:"required,oneof=e-mail sms in-app"`
	Status  string  `json:"status" binding:"required,oneof=FAILED SENDING SENT CANCELED"`
	ErrMsg  *string `json:"errMsg" binding:"omitempty,max=256"`
}

type NotificationRecipientStatusFilters struct {
	PageFilter
	Channels []string `json:"channel" binding:"unique,dive,oneof=e-mail sms in-app"`
	Statuses []string `json:"status" binding:"unique,dive,oneof=FAILED SENDING SENT CANCELED"`
}

type NotificationStatusResp struct {
	Status NotificationStatus `json:"status"`
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
