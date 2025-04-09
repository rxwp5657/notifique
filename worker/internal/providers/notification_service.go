package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/clients"
)

type addQueryParamsFn func(query url.Values)

type StatusFilters struct {
	NotificationId string
	Channels       []dto.NotificationChannel
	Statuses       []dto.NotificationStatus
}

type NotificationServiceProvider struct {
	clients.NotificationServiceClient
}

type paginatedApiInfo struct {
	Url            string
	AuthProvider   clients.AuthProvider
	AddQueryParams addQueryParamsFn
	Client         clients.NotificationServiceClient
}

func (p *NotificationServiceProvider) GetDistributionListRecipients(ctx context.Context, name string) ([]string, error) {

	url := fmt.Sprintf(
		string(clients.DistributionListEndpoint),
		p.NotificationServiceUrl, name)

	info := paginatedApiInfo{
		Url:            url,
		AuthProvider:   p.AuthProvider,
		AddQueryParams: nil,
		Client:         p.NotificationServiceClient,
	}

	recipients, err := consumePaginatedApi[string](ctx, info)

	if err != nil {
		return recipients, fmt.Errorf("error consuming paginated api - %w", err)
	}

	return recipients, nil
}

func (p *NotificationServiceProvider) GetNotificationTemplate(ctx context.Context, templateId string) (dto.NotificationTemplateDetails, error) {

	template := dto.NotificationTemplateDetails{}

	url := fmt.Sprintf(
		string(clients.NotificationTemplateEndpoint),
		p.NotificationServiceUrl, templateId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return template, fmt.Errorf("error creating request - %w", err)
	}

	err = p.AuthProvider.AddAuth(req)

	if err != nil {
		return template, fmt.Errorf("error adding auth to request - %w", err)
	}

	res, err := p.DoRequestWithBackoff(req, 0)

	if err != nil {
		return template, fmt.Errorf("error sending request - %w", err)
	}

	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&template); err != nil {
		return template, fmt.Errorf("error unmarshalling the template - %w", err)
	}

	return template, nil
}

func (p *NotificationServiceProvider) GetNotificationStatus(ctx context.Context, notificationId string) (dto.NotificationStatus, error) {

	url := fmt.Sprintf(
		string(clients.NotificationStatusEndpoint),
		p.NotificationServiceUrl, notificationId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	status := dto.NotificationStatus("")

	if err != nil {
		return status, fmt.Errorf("error creating request - %w", err)
	}

	err = p.AuthProvider.AddAuth(req)

	if err != nil {
		return status, fmt.Errorf("error adding auth to request - %w", err)
	}

	res, err := p.DoRequestWithBackoff(req, 0)

	if err != nil {
		return status, fmt.Errorf("error sending request - %w", err)
	}

	defer res.Body.Close()

	statusResp := dto.NotificationStatusResp{}

	if err := json.NewDecoder(res.Body).Decode(&statusResp); err != nil {
		return status, fmt.Errorf("error unmarshalling the response - %w", err)
	}

	return statusResp.Status, nil
}

func (p *NotificationServiceProvider) GetRecipientNotificationStatuses(ctx context.Context, filters StatusFilters) ([]dto.RecipientNotificationStatus, error) {

	statusUrl := fmt.Sprintf(
		string(clients.NotificationRecipientsStatusEndpoint),
		p.NotificationServiceUrl,
		filters.NotificationId)

	channelParams := func(query url.Values) {
		for _, channel := range filters.Channels {
			query.Add("channels", string(channel))
		}

		for _, status := range filters.Statuses {
			query.Add("statuses", string(status))
		}
	}

	info := paginatedApiInfo{
		Url:            statusUrl,
		AuthProvider:   p.AuthProvider,
		AddQueryParams: channelParams,
		Client:         p.NotificationServiceClient,
	}

	statuses, err := consumePaginatedApi[dto.RecipientNotificationStatus](ctx, info)

	if err != nil {
		return statuses, fmt.Errorf("error consuming paginated api - %v", err)
	}

	return statuses, nil
}

func consumePaginatedApi[T any](ctx context.Context, info paginatedApiInfo) ([]T, error) {

	data := []T{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.Url, nil)

	if err != nil {
		return data, fmt.Errorf("error creating request - %w", err)
	}

	err = info.AuthProvider.AddAuth(req)

	if err != nil {
		return data, fmt.Errorf("error adding auth to request - %w", err)
	}

	query := req.URL.Query()
	query.Add(clients.MaxResultsParamName, string(clients.MaxResults))

	if info.AddQueryParams != nil {
		info.AddQueryParams(query)
	}

	req.URL.RawQuery = query.Encode()

	for {

		res, err := info.Client.DoRequestWithBackoff(req, 0)

		if err != nil {
			return data, fmt.Errorf("error sending request - %w", err)
		}

		defer res.Body.Close()

		page := dto.Page[T]{}

		if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
			return data, fmt.Errorf("error unmarshalling the response - %w", err)
		}

		data = append(data, page.Data...)

		if page.NextToken == nil {
			break
		}

		query.Set(clients.NextTokenParamName, *page.NextToken)
		req.URL.RawQuery = query.Encode()
	}

	return data, nil
}

func NewNotificationServiceProvider(c clients.NotificationServiceClient) *NotificationServiceProvider {
	return &NotificationServiceProvider{
		NotificationServiceClient: c,
	}
}
