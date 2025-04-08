package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/clients"
)

type NotificationServiceSender struct {
	clients.NotificationServiceClient
}

func (s *NotificationServiceSender) SendNotifications(ctx context.Context, batch []dto.UserNotificationReq) error {

	body, err := json.Marshal(batch)

	if err != nil {
		return fmt.Errorf("error marshalling batch - %w", err)
	}

	url := fmt.Sprintf(
		string(clients.UsersNotificationsEndpoint),
		s.NotificationServiceUrl)

	return s.doRequestWithNoResponse(ctx, url, http.MethodPost, body)
}

func (s *NotificationServiceSender) UpdateNotificationStatus(ctx context.Context, log dto.NotificationStatusLog) error {
	body, err := json.Marshal(log)

	if err != nil {
		return fmt.Errorf("error marshalling batch - %w", err)
	}

	url := fmt.Sprintf(
		string(clients.NotificationStatusEndpoint),
		s.NotificationServiceUrl,
		log.NotificationId)

	return s.doRequestWithNoResponse(ctx, url, http.MethodPut, body)
}

func (s *NotificationServiceSender) UpdateRecipientNotificationStatus(ctx context.Context, notificationID string, batch []dto.RecipientNotificationStatus) error {

	body, err := json.Marshal(batch)

	if err != nil {
		return fmt.Errorf("error marshalling batch - %w", err)
	}

	url := fmt.Sprintf(
		string(clients.NotificationRecipientsStatusEndpoint),
		s.NotificationServiceUrl,
		notificationID)

	return s.doRequestWithNoResponse(ctx, url, http.MethodPost, body)
}

func (s *NotificationServiceSender) doRequestWithNoResponse(ctx context.Context, url, method string, body []byte) error {

	req, err := http.NewRequestWithContext(
		ctx, method,
		url, bytes.NewReader(body))

	if err != nil {
		return fmt.Errorf("error creating request - %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	err = s.AuthProvider.AddAuth(req)

	if err != nil {
		return fmt.Errorf("error adding auth to request - %w", err)
	}

	resp, err := s.DoRequestWithBackoff(req, 0)

	if err != nil {
		return fmt.Errorf("error sending request - %w", err)
	}

	defer resp.Body.Close()

	return nil
}

func NewNotificationServiceSender(c clients.NotificationServiceClient) *NotificationServiceSender {
	return &NotificationServiceSender{
		NotificationServiceClient: c,
	}
}
