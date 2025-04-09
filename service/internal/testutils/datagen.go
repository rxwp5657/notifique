package testutils

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/notifique/service/internal/dto"
	sdto "github.com/notifique/shared/dto"
)

const (
	GenericNotificationTemplateName = "Template name %s"
)

func MakeRecipients(numRecipients int) []string {

	recipients := make([]string, 0, numRecipients)

	for i := range cap(recipients) {
		recipients = append(recipients, strconv.Itoa(i))
	}

	return recipients
}

func MakeDistributionLists(numLists int) []dto.DistributionList {

	lists := make([]dto.DistributionList, 0, numLists)

	for i := range numLists {
		list := dto.DistributionList{
			Name:       fmt.Sprintf("Test List %d", i),
			Recipients: MakeRecipients(rand.Intn(10)),
		}

		lists = append(lists, list)
	}

	return lists
}

func MakeSummaries(lists []dto.DistributionList) []dto.DistributionListSummary {
	summaries := make([]dto.DistributionListSummary, 0, len(lists))

	for _, l := range lists {
		summary := dto.DistributionListSummary{
			Name:               l.Name,
			NumberOfRecipients: len(l.Recipients),
		}

		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	return summaries
}

func MakeTestNotificationRequestRawContents() sdto.NotificationReq {

	rawContents := &sdto.RawContents{
		Title:    "Notification 1",
		Contents: "Notification Contents 1",
	}

	testNofiticationReq := sdto.NotificationReq{
		RawContents:      rawContents,
		Topic:            "Testing",
		Priority:         "MEDIUM",
		DistributionList: nil,
		Recipients:       []string{"1234"},
		Channels:         []sdto.NotificationChannel{"in-app", "e-mail"},
	}

	return testNofiticationReq
}

func MakeTestNotificationRequestTemplateContents(templateId string, templateReq dto.NotificationTemplateReq) sdto.NotificationReq {

	variables := make([]sdto.TemplateVariableContents, 0, len(templateReq.Variables))

	testVariables := map[dto.TemplateVariableType]string{
		dto.Number:   "10",
		dto.String:   "test string",
		dto.Date:     "2025-01-19",
		dto.DateTime: "2025-01-19T00:00:00Z",
	}

	for _, v := range templateReq.Variables {
		variables = append(variables, sdto.TemplateVariableContents{
			Name:  v.Name,
			Value: testVariables[dto.TemplateVariableType(v.Type)],
		})
	}

	templateContents := &sdto.TemplateContents{
		Id:        templateId,
		Variables: variables,
	}

	testNofiticationReq := sdto.NotificationReq{
		TemplateContents: templateContents,
		Topic:            "Testing",
		Priority:         "MEDIUM",
		DistributionList: nil,
		Recipients:       []string{"1234"},
		Channels:         []sdto.NotificationChannel{"in-app", "e-mail"},
	}

	return testNofiticationReq
}

func MakeStrWithSize(size int) string {
	str, i := "", 0

	for i < size {
		str += "+"
		i += 1
	}

	return str
}

func MakeTestUserNotifications(numNotifications int, userId string) ([]dto.UserNotification, error) {
	testNotifications := make([]dto.UserNotification, 0, numNotifications)

	for i := range numNotifications {
		id, err := uuid.NewV7()

		if err != nil {
			return testNotifications, err
		}

		notification := dto.UserNotification{
			Id:        id.String(),
			Title:     fmt.Sprintf("Test title %d", i),
			Contents:  fmt.Sprintf("Test contents %d", i),
			Topic:     "Testing",
			CreatedAt: time.Now().Format(time.RFC3339Nano),
		}

		testNotifications = append(testNotifications, notification)
	}

	sort.Slice(testNotifications, func(i, j int) bool {
		return testNotifications[i].Id > testNotifications[j].Id
	})

	return testNotifications, nil
}

func MakeTestUserConfig(userId string) dto.UserConfig {
	cfg := dto.UserConfig{
		EmailConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
		SMSConfig:   dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
		InAppConfig: dto.ChannelConfig{OptIn: true, SnoozeUntil: nil},
	}

	return cfg
}

func MakeTestNotificationTemplateRequest() dto.NotificationTemplateReq {
	return dto.NotificationTemplateReq{
		Name:             "signed-in-notification",
		IsHtml:           true,
		TitleTemplate:    "Hi {user}!",
		ContentsTemplate: "Welcome to {app_name}!",
		Description:      "User has signed-in",
		Variables: []sdto.TemplateVariable{
			{
				Name:     "{user}",
				Type:     "STRING",
				Required: true,
			},
			{
				Name:     "{app_name}",
				Type:     "STRING",
				Required: true,
			},
		},
	}
}

func MakeTestNotificationTemplateRequests(numrequests int) []dto.NotificationTemplateReq {

	requests := make([]dto.NotificationTemplateReq, 0, numrequests)

	for i := range numrequests {
		req := MakeTestNotificationTemplateRequest()
		req.Name = fmt.Sprintf(GenericNotificationTemplateName, strconv.Itoa(i))
		requests = append(requests, req)
	}

	return requests
}

func MakeTestNotificationTemplateFilter() dto.NotificationTemplateFilters {
	return dto.NotificationTemplateFilters{
		PageFilter: sdto.PageFilter{
			NextToken:  nil,
			MaxResults: nil,
		},
		TemplateName: nil,
	}
}

func MakeTestNotificationTemplateInfoResp(numresps int) []dto.NotificationTemplateInfoResp {

	resps := make([]dto.NotificationTemplateInfoResp, 0, numresps)

	for i := range numresps {
		resp := dto.NotificationTemplateInfoResp{
			Id:          uuid.NewString(),
			Name:        fmt.Sprintf(GenericNotificationTemplateName, strconv.Itoa(i)),
			Description: fmt.Sprintf("Test Description %s", strconv.Itoa(i)),
		}
		resps = append(resps, resp)
	}

	return resps
}

func MakeNotificationSummary(req sdto.NotificationReq, id, userId string) dto.NotificationSummary {

	var contentsType dto.NotificationContentsType

	if req.RawContents != nil {
		contentsType = dto.Raw
	} else {
		contentsType = dto.Template
	}

	return dto.NotificationSummary{
		Id:           id,
		Topic:        req.Topic,
		ContentsType: contentsType,
		Priority:     req.Priority,
		Status:       sdto.Created,
		CreatedAt:    time.Now().Format(time.RFC3339Nano),
		CreatedBy:    userId,
	}
}

func StrPtr(s string) *string {
	return &s
}

func StatusPtr(status sdto.NotificationStatus) *sdto.NotificationStatus {
	return &status
}

func IntPtr(i int) *int {
	return &i
}

func MakeTestRecipientNotifcationStatus(recipients []string, channels []sdto.NotificationChannel, status sdto.NotificationStatus) []sdto.RecipientNotificationStatus {

	statuses := make([]sdto.RecipientNotificationStatus, 0, len(recipients))

	for _, recipient := range recipients {
		for _, channel := range channels {
			statuses = append(statuses, sdto.RecipientNotificationStatus{
				UserId:  recipient,
				Channel: string(channel),
				Status:  string(status),
				ErrMsg:  nil,
			})
		}
	}

	return statuses
}
