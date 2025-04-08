package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/controllers"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/service/internal/registry"
	"github.com/notifique/service/internal/testutils"
	r "github.com/notifique/service/internal/testutils/registry"
	sdto "github.com/notifique/shared/dto"
)

type NotificationRegistryTester interface {
	controllers.NotificationRegistry
	controllers.NotificationTemplateRegistry
	r.ContainerTester
}

func TestNotificationRegistryPostgres(t *testing.T) {

	ctx := context.Background()
	tester, close, err := r.NewPostgresIntegrationTester(ctx)

	if err != nil {
		t.Fatal("failed to init postgres tester - ", err)
	}

	defer close()

	testCreateNotification(ctx, t, tester)
	testUpdateNotificationStatus(ctx, t, tester)
	testDeleteNotification(ctx, t, tester)
	testGetNotifications(ctx, t, tester)
	testGetNotification(ctx, t, tester)
	testGetRecipientNotificationStatuses(ctx, t, tester)
}

func TestNotificationRegistryDynamo(t *testing.T) {
	ctx := context.Background()
	tester, close, err := r.NewDynamoRegistryTester(ctx)

	if err != nil {
		t.Fatal("failed to init dynamo tester - ", err)
	}

	defer close()

	testCreateNotification(ctx, t, tester)
	testUpdateNotificationStatus(ctx, t, tester)
	testDeleteNotification(ctx, t, tester)
	testGetNotifications(ctx, t, tester)
	testGetNotification(ctx, t, tester)
	testGetRecipientNotificationStatuses(ctx, t, tester)
}

func testCreateNotification(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {

	userId := "1234"
	defer r.Clear(ctx, t, nt)

	// Create template for template-based notification tests
	templateReq := dto.NotificationTemplateReq{
		Name:             "signed-in-notification",
		TitleTemplate:    "Hi {user}!",
		ContentsTemplate: "Welcome {optional} to {app_name}!",
		Description:      "User has signed-in",
		Variables: []sdto.TemplateVariable{
			{
				Name:       "{user}",
				Type:       "STRING",
				Required:   true,
				Validation: testutils.StrPtr("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"),
			},
			{
				Name:     "{app_name}",
				Type:     "STRING",
				Required: true,
			},
			{
				Name:     "{optional}",
				Type:     "NUMBER",
				Required: false,
			},
		},
	}

	templateResp, err := nt.SaveTemplate(ctx, userId, templateReq)

	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		request sdto.NotificationReq
	}{
		{
			name:    "Create notification with raw contents",
			request: testutils.MakeTestNotificationRequestRawContents(),
		},
		{
			name: "Create notification with template contents",
			request: sdto.NotificationReq{
				TemplateContents: &sdto.TemplateContents{
					Id: templateResp.Id,
					Variables: []sdto.TemplateVariableContents{
						{
							Name:  "{user}",
							Value: "550e8400-e29b-41d4-a716-446655440000",
						},
						{
							Name:  "{app_name}",
							Value: "Test App",
						},
						{
							Name:  "{optional}",
							Value: "42",
						},
					},
				},
				Topic:      "template-test",
				Priority:   "HIGH",
				Recipients: []string{"user1", "user2"},
				Channels:   []sdto.NotificationChannel{"e-mail", "sms"},
			},
		},
		{
			name: "Create notification with invalid template variable",
			request: sdto.NotificationReq{
				TemplateContents: &sdto.TemplateContents{
					Id: templateResp.Id,
					Variables: []sdto.TemplateVariableContents{
						{
							Name:  "{user}",
							Value: "invalid-uuid",
						},
						{
							Name:  "{app_name}",
							Value: "Test App",
						},
					},
				},
				Topic:      "invalid-template",
				Priority:   "LOW",
				Recipients: []string{"user1", "user2"},
				Channels:   []sdto.NotificationChannel{"e-mail", "sms"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notificationId, err := nt.SaveNotification(ctx, userId, tt.request)

			assert.Nil(t, err)
			assert.Nil(t, uuid.Validate(notificationId))

			// Verify stored notification
			storedNotification, err := nt.GetNotification(ctx, notificationId)
			if err != nil {
				t.Fatal(err)
			}

			if tt.request.RawContents != nil {
				assert.Equal(t, tt.request.RawContents.Title, storedNotification.RawContents.Title)
				assert.Equal(t, tt.request.RawContents.Contents, storedNotification.RawContents.Contents)
			}

			if tt.request.TemplateContents != nil {
				assert.Equal(t, tt.request.TemplateContents.Id, storedNotification.TemplateContents.Id)
				assert.ElementsMatch(t, tt.request.TemplateContents.Variables, storedNotification.TemplateContents.Variables)
			}

			assert.Equal(t, tt.request.Topic, storedNotification.Topic)
			assert.Equal(t, tt.request.Priority, storedNotification.Priority)
			assert.Equal(t, tt.request.DistributionList, storedNotification.DistributionList)
			assert.ElementsMatch(t, tt.request.Recipients, storedNotification.Recipients)
			assert.ElementsMatch(t, tt.request.Channels, storedNotification.Channels)

			status, err := nt.GetNotificationStatus(ctx, notificationId)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, sdto.Created, status)
		})
	}
}

func testUpdateNotificationStatus(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {

	user := "1234"
	testNofiticationReq := testutils.MakeTestNotificationRequestRawContents()
	notificationId, err := nt.SaveNotification(ctx, user, testNofiticationReq)

	if err != nil {
		t.Fatal(err)
	}

	defer r.Clear(ctx, t, nt)

	t.Run("Should be able to update the notification status", func(t *testing.T) {

		log := sdto.NotificationStatusLog{
			NotificationId: notificationId,
			Status:         sdto.Queued,
			ErrorMsg:       nil,
		}

		err := nt.UpdateNotificationStatus(ctx, log)

		assert.Nil(t, err)

		status, err := nt.GetNotificationStatus(ctx, notificationId)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, sdto.Queued, status)
	})

	t.Run("Should handle updating status of non-existent notification", func(t *testing.T) {
		nonExistentId := uuid.NewString()
		log := sdto.NotificationStatusLog{
			NotificationId: nonExistentId,
			Status:         sdto.Queued,
			ErrorMsg:       nil,
		}

		err := nt.UpdateNotificationStatus(ctx, log)
		assert.ErrorAs(t, err, &internal.EntityNotFound{Id: nonExistentId, Type: registry.NotificationType})
	})
}

func testDeleteNotification(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {

	user := "1234"

	testNotifications := map[sdto.NotificationStatus]string{
		sdto.Created: "",
		sdto.Queued:  "",
		sdto.Sending: "",
		sdto.Sent:    "",
		sdto.Failed:  "",
	}

	for status := range testNotifications {
		testNofiticationReq := testutils.MakeTestNotificationRequestRawContents()
		notificationId, err := nt.SaveNotification(ctx, user, testNofiticationReq)

		if err != nil {
			t.Fatal(err)
		}

		err = nt.UpdateNotificationStatus(ctx, sdto.NotificationStatusLog{
			NotificationId: notificationId,
			Status:         status,
		})

		if err != nil {
			t.Fatal(err)
		}

		testNotifications[status] = notificationId
	}

	defer r.Clear(ctx, t, nt)

	tests := []struct {
		name           string
		notificationId string
		expectedError  error
	}{
		{
			name:           "Can delete a notification in CREATED state",
			notificationId: testNotifications[sdto.Created],
			expectedError:  nil,
		},
		{
			name:           "Can delete a notification in FAILED state",
			notificationId: testNotifications[sdto.Failed],
			expectedError:  nil,
		},
		{
			name:           "Can delete a notification in SENT state",
			notificationId: testNotifications[sdto.Sent],
			expectedError:  nil,
		},
		{
			name:           "Should fail deleting a notification on QUEUED state",
			notificationId: testNotifications[sdto.Queued],
			expectedError: internal.InvalidNotificationStatus{
				Id:     testNotifications[sdto.Queued],
				Status: string(sdto.Queued),
			},
		},
		{
			name:           "Should fail deleting a notification on SENDING state",
			notificationId: testNotifications[sdto.Sending],
			expectedError: internal.InvalidNotificationStatus{
				Id:     testNotifications[sdto.Sending],
				Status: string(sdto.Sending),
			},
		},
		{
			name:           "Should be able to delete a notification that doesn't exist",
			notificationId: uuid.NewString(),
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualErr := nt.DeleteNotification(ctx, tt.notificationId)
			assert.Equal(t, tt.expectedError, actualErr)

			if tt.expectedError == nil {
				_, err := nt.GetNotificationStatus(ctx, tt.notificationId)
				assert.ErrorAs(t, err, &internal.EntityNotFound{Id: tt.notificationId, Type: registry.NotificationType})
			}
		})
	}
}

func testGetNotifications(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {

	user := "1234"
	testNotificationSummaries := map[string]dto.NotificationSummary{}

	templateReq := testutils.MakeTestNotificationTemplateRequest()

	templateResp, err := nt.SaveTemplate(ctx, user, templateReq)

	if err != nil {
		t.Fatal(err)
	}

	for i := range 4 {
		var testNofiticationReq sdto.NotificationReq

		if i%2 == 0 {
			testNofiticationReq = testutils.MakeTestNotificationRequestRawContents()
		} else {
			testNofiticationReq = testutils.MakeTestNotificationRequestTemplateContents(
				templateResp.Id,
				templateReq,
			)
		}

		notificationId, err := nt.SaveNotification(ctx, user, testNofiticationReq)

		if err != nil {
			t.Fatal(err)
		}

		summary := testutils.MakeNotificationSummary(testNofiticationReq, notificationId, user)
		testNotificationSummaries[notificationId] = summary
	}

	areSummariesEqual := func(t *testing.T, a, b dto.NotificationSummary) {
		assert.Equal(t, a.Id, b.Id)
		assert.Equal(t, a.Topic, b.Topic)
		assert.Equal(t, a.Priority, b.Priority)
		assert.Equal(t, a.Status, b.Status)
		assert.Equal(t, a.ContentsType, b.ContentsType)
		assert.Equal(t, a.CreatedBy, b.CreatedBy)
	}

	defer r.Clear(ctx, t, nt)

	t.Run("Should be able to get notifications", func(t *testing.T) {

		filters := sdto.PageFilter{
			NextToken:  nil,
			MaxResults: nil,
		}

		page, err := nt.GetNotifications(ctx, filters)

		if err != nil {
			t.Fatal(err)
		}

		assert.Nil(t, page.NextToken)
		assert.Nil(t, page.PrevToken)
		assert.Equal(t, len(testNotificationSummaries), page.ResultCount)

		for _, s := range page.Data {
			areSummariesEqual(t, s, testNotificationSummaries[s.Id])
		}
	})

	t.Run("Should be able to get notifications with pagination", func(t *testing.T) {

		filters := sdto.PageFilter{
			NextToken:  nil,
			MaxResults: testutils.IntPtr(1),
		}

		summaries := make([]dto.NotificationSummary, 0, len(testNotificationSummaries))

		for {
			page, err := nt.GetNotifications(ctx, filters)

			if err != nil {
				t.Fatal(err)
			}

			summaries = append(summaries, page.Data...)

			if page.NextToken == nil {
				break
			}

			filters.NextToken = page.NextToken
		}

		for _, s := range summaries {
			areSummariesEqual(t, s, testNotificationSummaries[s.Id])
		}
	})
}

func testGetNotification(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {
	user := "1234"

	templateReq := testutils.MakeTestNotificationTemplateRequest()
	templateResp, err := nt.SaveTemplate(ctx, user, templateReq)

	if err != nil {
		t.Fatal(err)
	}

	defer r.Clear(ctx, t, nt)

	t.Run("Should be able to retrieve a notification with RawContents", func(t *testing.T) {
		notificationReq := testutils.MakeTestNotificationRequestRawContents()
		notificationId, err := nt.SaveNotification(ctx, user, notificationReq)

		if err != nil {
			t.Fatal(err)
		}

		notification, err := nt.GetNotification(ctx, notificationId)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, notificationReq.RawContents.Title, notification.RawContents.Title)
		assert.Equal(t, notificationReq.RawContents.Contents, notification.RawContents.Contents)
		assert.Equal(t, notificationReq.Topic, notification.Topic)
		assert.Equal(t, notificationReq.Priority, notification.Priority)
		assert.Equal(t, notificationReq.DistributionList, notification.DistributionList)
		assert.ElementsMatch(t, notificationReq.Recipients, notification.Recipients)
		assert.ElementsMatch(t, notificationReq.Channels, notification.Channels)
	})

	t.Run("Should be able to retrieve a notification with TemplateContents", func(t *testing.T) {
		notificationReq := testutils.MakeTestNotificationRequestTemplateContents(
			templateResp.Id,
			templateReq,
		)
		notificationId, err := nt.SaveNotification(ctx, user, notificationReq)

		if err != nil {
			t.Fatal(err)
		}

		notification, err := nt.GetNotification(ctx, notificationId)

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, notificationReq.TemplateContents.Id, notification.TemplateContents.Id)
		assert.ElementsMatch(t, notificationReq.TemplateContents.Variables, notification.TemplateContents.Variables)
		assert.Equal(t, notificationReq.Topic, notification.Topic)
		assert.Equal(t, notificationReq.Priority, notification.Priority)
		assert.Equal(t, notificationReq.DistributionList, notification.DistributionList)
		assert.ElementsMatch(t, notificationReq.Recipients, notification.Recipients)
		assert.ElementsMatch(t, notificationReq.Channels, notification.Channels)
	})

	t.Run("Should return EntityNotFound if the notification doesn't exist", func(t *testing.T) {
		nonExistentId := uuid.NewString()
		_, err := nt.GetNotification(ctx, nonExistentId)

		assert.ErrorAs(t, err, &internal.EntityNotFound{Id: nonExistentId, Type: registry.NotificationType})
	})

	t.Run("Should be able to retrieve a notification with TemplateContents and no variables", func(t *testing.T) {
		// Create simple template with no variables
		templateReq := dto.NotificationTemplateReq{
			Name:             "simple-template",
			TitleTemplate:    "Simple Title",
			ContentsTemplate: "Simple Contents",
			Description:      "Template with no variables",
			Variables:        []sdto.TemplateVariable{},
		}

		templateResp, err := nt.SaveTemplate(ctx, user, templateReq)
		if err != nil {
			t.Fatal(err)
		}

		// Create notification using template
		notificationReq := sdto.NotificationReq{
			TemplateContents: &sdto.TemplateContents{
				Id:        templateResp.Id,
				Variables: []sdto.TemplateVariableContents{},
			},
			Topic:      "simple-template-test",
			Priority:   "LOW",
			Recipients: []string{"user1"},
			Channels:   []sdto.NotificationChannel{"e-mail"},
		}

		notificationId, err := nt.SaveNotification(ctx, user, notificationReq)
		if err != nil {
			t.Fatal(err)
		}

		// Retrieve and verify notification
		notification, err := nt.GetNotification(ctx, notificationId)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, notificationReq.TemplateContents.Id, notification.TemplateContents.Id)
		assert.Empty(t, notification.TemplateContents.Variables)
		assert.Equal(t, notificationReq.Topic, notification.Topic)
		assert.Equal(t, notificationReq.Priority, notification.Priority)
		assert.ElementsMatch(t, notificationReq.Recipients, notification.Recipients)
		assert.ElementsMatch(t, notificationReq.Channels, notification.Channels)
	})
}

func testGetRecipientNotificationStatuses(ctx context.Context, t *testing.T, nt NotificationRegistryTester) {
	createdBy := "1234"

	recipients := testutils.MakeRecipients(2)
	channels := []sdto.NotificationChannel{"e-mail", "sms"}

	testNotificationReq := testutils.MakeTestNotificationRequestRawContents()
	testNotificationReq.Recipients = recipients
	testNotificationReq.Channels = channels

	notificationId, err := nt.SaveNotification(ctx, createdBy, testNotificationReq)

	if err != nil {
		t.Fatal(err)
	}

	firstStatuses := testutils.MakeTestRecipientNotifcationStatus(recipients, channels, sdto.Sending)
	statuses := testutils.MakeTestRecipientNotifcationStatus(recipients, channels, sdto.Sent)

	err = nt.UpsertRecipientNotificationStatuses(ctx, notificationId, firstStatuses)

	if err != nil {
		t.Fatal(err)
	}

	err = nt.UpsertRecipientNotificationStatuses(ctx, notificationId, statuses)

	if err != nil {
		t.Fatal(err)
	}

	defer r.Clear(ctx, t, nt)

	t.Run("Should be able to retrieve the latest user notification statuses", func(t *testing.T) {

		filters := sdto.NotificationRecipientStatusFilters{
			PageFilter: sdto.PageFilter{},
			Channels:   []string{},
		}

		page, err := nt.GetRecipientNotificationStatuses(ctx, notificationId, filters)

		if err != nil {
			t.Fatal(err)
		}

		assert.ElementsMatch(t, statuses, page.Data)
	})

	t.Run("Should be able to retrieve the latest user notification statuses with pagination", func(t *testing.T) {
		filters := sdto.NotificationRecipientStatusFilters{
			PageFilter: sdto.PageFilter{
				MaxResults: testutils.IntPtr(1),
			},
			Channels: []string{},
		}

		receivedStatuses := make([]sdto.RecipientNotificationStatus, 0, len(statuses))

		for {
			page, err := nt.GetRecipientNotificationStatuses(ctx, notificationId, filters)

			if err != nil {
				t.Fatal(err)
			}

			receivedStatuses = append(receivedStatuses, page.Data...)

			if page.NextToken == nil {
				break
			}

			filters.PageFilter.NextToken = page.NextToken
		}

		assert.ElementsMatch(t, statuses, receivedStatuses)
	})

	t.Run("Should be able to retrieve the latest user notification statuses with channel filter", func(t *testing.T) {
		filters := sdto.NotificationRecipientStatusFilters{
			PageFilter: sdto.PageFilter{},
			Channels:   []string{"e-mail"},
		}

		page, err := nt.GetRecipientNotificationStatuses(ctx, notificationId, filters)

		if err != nil {
			t.Fatal(err)
		}

		expectedStatuses := []sdto.RecipientNotificationStatus{}

		for _, status := range statuses {
			if status.Channel == "e-mail" {
				expectedStatuses = append(expectedStatuses, status)
			}
		}

		assert.ElementsMatch(t, expectedStatuses, page.Data)
	})

	t.Run("Should be able to retrieve the latest user notification statuses with status filter", func(t *testing.T) {
		filters := sdto.NotificationRecipientStatusFilters{
			PageFilter: sdto.PageFilter{},
			Statuses:   []string{"SENT"},
		}

		page, err := nt.GetRecipientNotificationStatuses(ctx, notificationId, filters)

		if err != nil {
			t.Fatal(err)
		}

		expectedStatuses := []sdto.RecipientNotificationStatus{}

		for _, status := range statuses {
			if status.Status == "SENT" {
				expectedStatuses = append(expectedStatuses, status)
			}
		}

		assert.ElementsMatch(t, expectedStatuses, page.Data)
	})

	t.Run("Should be able to retrieve the latest user notification statuses with status and channel filter", func(t *testing.T) {
		filters := sdto.NotificationRecipientStatusFilters{
			PageFilter: sdto.PageFilter{},
			Channels:   []string{string(sdto.Email)},
			Statuses:   []string{string(sdto.Sent)},
		}

		page, err := nt.GetRecipientNotificationStatuses(ctx, notificationId, filters)

		if err != nil {
			t.Fatal(err)
		}

		expectedStatuses := []sdto.RecipientNotificationStatus{}

		for _, status := range statuses {
			if status.Status == string(sdto.Sent) && status.Channel == string(sdto.Email) {
				expectedStatuses = append(expectedStatuses, status)
			}
		}

		assert.ElementsMatch(t, expectedStatuses, page.Data)
	})
}
