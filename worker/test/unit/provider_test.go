package unit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/notifique/shared/dto"
	"github.com/notifique/worker/internal/clients"
	"github.com/notifique/worker/internal/providers"
	servers_test "github.com/notifique/worker/internal/testutils/servers"
	"github.com/stretchr/testify/assert"
)

func TestNotificationServiceProvider(t *testing.T) {

	responses := map[string]any{}

	recipients := []string{"user1@test.com", "user2@test.com"}

	template := dto.NotificationTemplateDetails{
		Id:               "test-template",
		Name:             "Test Template",
		Description:      "Test template description",
		TitleTemplate:    "Hello {{name}}",
		ContentsTemplate: "Hello {{name}}",
		CreatedAt:        "2023-01-01T00:00:00Z",
		CreatedBy:        "test-user",
		Variables: []dto.TemplateVariable{{
			Name:     "name",
			Type:     "STRING",
			Required: true,
		}},
	}

	notificationStatuses := []dto.RecipientNotificationStatus{
		{
			UserId:  "user1@test.com",
			Channel: "in-app",
			Status:  string(dto.Sending),
			ErrMsg:  nil,
		}, {
			UserId:  "user2@test.com",
			Channel: "e-mail",
			Status:  string(dto.Sending),
			ErrMsg:  nil,
		},
	}

	responses["/distribution-lists/test-list/recipients"] = recipients
	responses["/notifications/templates/test-template"] = template
	responses["/notifications/test-notification-id/recipients/statuses"] = notificationStatuses

	setupTestServer := func(url string, handlerFunc func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *providers.NotificationServiceProvider) {
		server := servers_test.NewTestServer(url, handlerFunc)

		provider := providers.NewNotificationServiceProvider(clients.NotificationServiceClient{
			AuthProvider:           clients.NoAuth,
			NotificationServiceUrl: clients.NotificationServiceUrl(server.URL),
			NumRetries:             0,
			BaseDelay:              0,
			MaxDelay:               0,
		})

		return server, provider
	}

	ctx := context.Background()

	t.Run("Can get distribution list recipients", func(t *testing.T) {
		server, provider := setupTestServer("GET /distribution-lists/{id}/recipients",
			servers_test.MakeDistributionListHandler(responses))

		defer server.Close()

		dlRecipients, err := provider.GetDistributionListRecipients(ctx, "test-list")

		if err != nil {
			t.Fatalf("error getting recipients - %v", err)
		}

		assert.ElementsMatch(t, recipients, dlRecipients)
	})

	t.Run("Can get template details", func(t *testing.T) {
		server, provider := setupTestServer("GET /notifications/templates/{id}",
			servers_test.MakeNotificationTemplateHandler(responses))

		defer server.Close()
		templateDetails, err := provider.GetNotificationTemplate(ctx, "test-template")

		if err != nil {
			t.Fatalf("error getting template - %v", err)
		}

		assert.Equal(t, template, templateDetails)
	})

	t.Run("Can get user notification statuses", func(t *testing.T) {
		server, provider := setupTestServer("GET /notifications/{id}/recipients/statuses",
			servers_test.MakeNotificationStatusHandler(responses))

		defer server.Close()

		filters := providers.StatusFilters{
			NotificationId: "test-notification-id",
			Channels:       []dto.NotificationChannel{},
		}

		notificationStatuses, err := provider.GetRecipientNotificationStatuses(ctx, filters)

		if err != nil {
			t.Fatalf("error getting notification statuses - %v", err)
		}

		assert.ElementsMatch(t, notificationStatuses, notificationStatuses)
	})

	t.Run("Can get user notification statuses with channel filter", func(t *testing.T) {
		server, provider := setupTestServer("GET /notifications/{id}/recipients/statuses",
			servers_test.MakeNotificationStatusHandler(responses))

		defer server.Close()

		filters := providers.StatusFilters{
			NotificationId: "test-notification-id",
			Channels:       []dto.NotificationChannel{dto.Email},
		}

		notificationStatuses, err := provider.GetRecipientNotificationStatuses(ctx, filters)

		if err != nil {
			t.Fatalf("error getting notification statuses - %v", err)
		}

		expectedStatuses := []dto.RecipientNotificationStatus{}

		for _, status := range notificationStatuses {
			if status.Channel == "e-mail" {
				expectedStatuses = append(expectedStatuses, status)
			}
		}

		assert.ElementsMatch(t, expectedStatuses, notificationStatuses)
	})
}
