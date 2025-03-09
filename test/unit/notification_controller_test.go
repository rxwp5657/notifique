package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	di "github.com/notifique/internal/di"
	"github.com/notifique/internal/registry"
	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/dto"
	"github.com/notifique/internal/testutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const notificationsUrl = "/notifications"

func TestNotificationController(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	testApp, err := di.InjectMockedBackend(context.TODO(), controller)
	if err != nil {
		t.Fatalf("failed to create mocked backend - %v", err)
	}

	testCreateNotification(t, testApp.Engine, *testApp)
	testDeleteNotification(t, testApp.Engine, *testApp)
	testCancelNotificationDelivery(t, testApp.Engine, *testApp)
	testGetNotifications(t, testApp.Engine, *testApp)
}

func testCreateNotification(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	mock.Publisher.
		EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	mock.Cache.
		EXPECT().
		UpdateNotificationStatus(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(2)

	registryMock := mock.Registry.MockNotificationRegistry
	userId := "1234"

	createNotification := func(notificationReq dto.NotificationReq) *httptest.ResponseRecorder {
		body, _ := json.Marshal(notificationReq)
		reader := bytes.NewReader(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, notificationsUrl, reader)
		req.Header.Add("userId", userId)
		e.ServeHTTP(w, req)
		return w
	}

	randomTemplateId := uuid.NewString()

	tests := []struct {
		name           string
		setupMock      func()
		modifyRequest  func(req dto.NotificationReq) dto.NotificationReq
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Can create new notifications with raw contents",
			setupMock: func() {
				registryMock.EXPECT().
					SaveNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uuid.NewString(), nil)
			},
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.RawContents = &dto.RawContents{
					Title:    "Test Title",
					Contents: "Test Contents",
				}
				req.TemplateContents = nil
				return req
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "Can create new notifications with template contents",
			setupMock: func() {
				registryMock.EXPECT().
					GetTemplateVariables(gomock.Any(), gomock.Any()).
					Return([]dto.TemplateVariable{
						{Name: "{user}", Type: "STRING", Required: true},
						{Name: "{date}", Type: "DATE", Required: true},
					}, nil)
				registryMock.EXPECT().
					SaveNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uuid.NewString(), nil)
			},
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.TemplateContents = &dto.TemplateContents{
					Id: uuid.NewString(),
					Variables: []dto.TemplateVariableContents{
						{Name: "{user}", Value: "John"},
						{Name: "{date}", Value: "2024-01-01"},
					},
				}
				req.RawContents = nil
				return req
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "Should fail when template doesn't exist",
			setupMock: func() {
				registryMock.
					EXPECT().
					GetTemplateVariables(gomock.Any(), gomock.Any()).
					Return(nil, server.EntityNotFound{
						Id: randomTemplateId, Type: registry.NotificationTemplateType,
					})
			},
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.TemplateContents = &dto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []dto.TemplateVariableContents{{Name: "{user}", Value: "John"}},
				}
				req.RawContents = nil
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Template not found",
		},
		{
			name: "Should fail when template variable has invalid type",
			setupMock: func() {
				registryMock.EXPECT().
					GetTemplateVariables(gomock.Any(), gomock.Any()).
					Return([]dto.TemplateVariable{
						{Name: "{date}", Type: "DATE", Required: true},
					}, nil)
			},
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.TemplateContents = &dto.TemplateContents{
					Id: uuid.NewString(),
					Variables: []dto.TemplateVariableContents{{
						Name:  "{date}",
						Value: "not-a-date"}},
				}
				req.RawContents = nil
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "not-a-date is not a valid date",
		},
		{
			name: "Should fail when supplying a non-existing template variable",
			setupMock: func() {
				registryMock.EXPECT().
					GetTemplateVariables(gomock.Any(), gomock.Any()).
					Return([]dto.TemplateVariable{
						{Name: "{user}", Type: "STRING", Required: true},
					}, nil)
			},
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.TemplateContents = &dto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []dto.TemplateVariableContents{{Name: "{invalid}", Value: "value"}},
				}
				req.RawContents = nil
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "{invalid} is not a template variable",
		},
		{
			name: "Should fail when required template variable is missing",
			setupMock: func() {
				registryMock.EXPECT().
					GetTemplateVariables(gomock.Any(), gomock.Any()).
					Return([]dto.TemplateVariable{
						{Name: "{user}", Type: "STRING", Required: true},
						{Name: "{date}", Type: "DATE", Required: true},
					}, nil)
			},
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.TemplateContents = &dto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []dto.TemplateVariableContents{{Name: "{user}", Value: "John"}},
				}
				req.RawContents = nil
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "template variable {date} not found",
		},
		{
			name: "Should fail when template variable fails validation",
			setupMock: func() {
				pattern := "^[A-Z]+$"
				registryMock.EXPECT().
					GetTemplateVariables(gomock.Any(), gomock.Any()).
					Return([]dto.TemplateVariable{
						{Name: "{user}", Type: "STRING", Required: true, Validation: &pattern},
					}, nil)
			},
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.TemplateContents = &dto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []dto.TemplateVariableContents{{Name: "{user}", Value: "lowercase"}},
				}
				req.RawContents = nil
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "lowercase failed regex validation ^[A-Z]+$",
		},
		{
			name: "Should fail when neither raw contents nor template contents are provided",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.RawContents = nil
				req.TemplateContents = nil
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Error:Field validation for 'RawContents' failed on the 'required_without' tag",
		},
		{
			name: "Should fail when both raw contents and template contents are provided",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.RawContents = &dto.RawContents{
					Title:    "Test Title",
					Contents: "Test Contents",
				}
				req.TemplateContents = &dto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []dto.TemplateVariableContents{{Name: "{user}", Value: "John"}},
				}
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Error:Field validation for 'RawContents' failed on the 'excluded_with' tag",
		},
		{
			name: "Should fail if the channel is not supported",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.Channels = append(req.Channels, "INVALID")
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `NotificationReq.Channels[2]' Error:Field validation for 'Channels[2]' failed on the 'oneof' tag`,
		},
		{
			name: "Should fail if the title exceeds the limits",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.RawContents.Title = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.RawContents.Title' Error:Field validation for 'Title' failed on the 'max' tag`,
		},
		{
			name: "Should fail if the title is empty",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.RawContents.Title = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.RawContents.Title' Error:Field validation for 'Title' failed on the 'required' tag`,
		},
		{
			name: "Should fail if the contents are empty",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.RawContents.Contents = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.RawContents.Contents' Error:Field validation for 'Contents' failed on the 'required' tag`,
		},
		{
			name: "Should fail if the contents exceeds the limits",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.RawContents.Contents = testutils.MakeStrWithSize(1025)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.RawContents.Contents' Error:Field validation for 'Contents' failed on the 'max' tag`,
		},
		{
			name: "Should fail if the topic exceeds the limits",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.Topic = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Topic' Error:Field validation for 'Topic' failed on the 'max' tag`,
		},
		{
			name: "Should fail if the topic is empty",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.Topic = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Topic' Error:Field validation for 'Topic' failed on the 'required' tag`,
		},
		{
			name: "Should fail if there are duplicated recipients",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.Recipients = append(req.Recipients, req.Recipients...)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Recipients' Error:Field validation for 'Recipients' failed on the 'unique' tag`,
		},
		{
			name: "Should fail if the priority is not supported",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.Priority = "Test"
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Priority' Error:Field validation for 'Priority' failed on the 'oneof' tag`,
		},
		{
			name: "Should fail if the image url is not an url",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				url := "not an url"
				req.Image = &url
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Image' Error:Field validation for 'Image' failed on the 'uri' tag`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			req := testutils.MakeTestNotificationRequestRawContents()
			req = tt.modifyRequest(req)
			w := createNotification(req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], tt.expectedError)
			}
		})
	}
}

func testDeleteNotification(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	userId := "1234"
	registryMock := mock.Registry.MockNotificationRegistry

	deleteNotification := func(notificationId string) *httptest.ResponseRecorder {
		url := fmt.Sprintf("%s/%s", notificationsUrl, notificationId)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.Header.Add("userId", userId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name           string
		notificationId string
		setupMock      func()
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Can delete a notification",
			notificationId: uuid.NewString(),
			expectedStatus: http.StatusNoContent,
			setupMock: func() {
				registryMock.
					EXPECT().
					DeleteNotification(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:           "Should fail if the id is not an uuid",
			notificationId: "not an uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'NotificationUriParams.NotificationId' Error:Field validation for 'NotificationId' failed on the 'uuid' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			w := deleteNotification(tt.notificationId)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], tt.expectedError)
			}
		})
	}
}

func testCancelNotificationDelivery(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	userId := "1234"

	cancelDelivery := func(notificationId string) *httptest.ResponseRecorder {
		url := fmt.Sprintf("%s/%s/cancel", notificationsUrl, notificationId)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, url, nil)
		req.Header.Add("userId", userId)
		e.ServeHTTP(w, req)
		return w
	}

	notificationId := uuid.NewString()
	createdStatus := dto.Created
	canceledStatus := dto.Canceled

	expectedStatusLog := controllers.NotificationStatusLog{
		NotificationId: notificationId,
		Status:         canceledStatus,
	}

	tests := []struct {
		name           string
		notificationId string
		setupMock      func()
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Can cancel a notification with CREATED status (with status retrieved from the cache)",
			notificationId: notificationId,
			expectedStatus: 204,
			setupMock: func() {
				mock.Cache.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(testutils.StatusPtr(createdStatus), nil).
					Times(1)

				mock.Cache.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), expectedStatusLog).
					Return(nil).
					Times(1)

				mock.Registry.MockNotificationRegistry.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), expectedStatusLog).
					Return(nil).
					Times(1)
			},
		},
		{
			name:           "Can cancel a notification with CREATED status (with status retrieved from the db)",
			notificationId: notificationId,
			expectedStatus: 204,
			setupMock: func() {
				mock.Cache.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(nil, nil).
					Times(1)

				mock.Registry.MockNotificationRegistry.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(createdStatus, nil).
					Times(1)

				mock.Cache.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), expectedStatusLog).
					Return(nil).
					Times(1)

				mock.Registry.MockNotificationRegistry.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), expectedStatusLog).
					Return(nil).
					Times(1)
			},
		},
		{
			name:           "Should fail if the status of the notification is SENDING (status from cache)",
			notificationId: notificationId,
			expectedStatus: 400,
			expectedError:  "Notification is being sent",
			setupMock: func() {
				sendingStatus := testutils.StatusPtr(dto.Sending)
				mock.Cache.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(sendingStatus, nil).
					Times(1)
			},
		},
		{
			name:           "Should fail if the status of the notification is SENT (status from cache)",
			notificationId: notificationId,
			expectedStatus: 400,
			expectedError:  "Notification has been sent",
			setupMock: func() {
				sentStatus := testutils.StatusPtr(dto.Sent)
				mock.Cache.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(sentStatus, nil).
					Times(1)
			},
		},
		{
			name:           "Should fail if the status of the notification is SENDING (status from db)",
			notificationId: notificationId,
			expectedStatus: 400,
			expectedError:  "Notification is being sent",
			setupMock: func() {
				sendingStatus := dto.Sending
				mock.Cache.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(nil, nil).
					Times(1)

				mock.Registry.MockNotificationRegistry.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(sendingStatus, nil).
					Times(1)
			},
		},
		{
			name:           "Should fail if the status of the notification is SENT (status from db)",
			notificationId: notificationId,
			expectedStatus: 400,
			expectedError:  "Notification has been sent",
			setupMock: func() {
				sentStatus := dto.Sent
				mock.Cache.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(nil, nil).
					Times(1)

				mock.Registry.MockNotificationRegistry.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(sentStatus, nil).
					Times(1)
			},
		},
		{
			name:           "Should fail if we can't update the cache",
			notificationId: notificationId,
			expectedStatus: 500,
			setupMock: func() {
				mock.Cache.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(testutils.StatusPtr(createdStatus), nil).
					Times(1)

				mock.Cache.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), expectedStatusLog).
					Return(errors.New("some failure")).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			w := cancelDelivery(tt.notificationId)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], tt.expectedError)
			}
		})
	}
}

func testGetNotifications(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	userId := "1234"
	registryMock := mock.Registry.MockNotificationRegistry
	getNotifications := func(filters dto.PageFilter) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, notificationsUrl, nil)
		req.Header.Add("userId", userId)
		testutils.AddPaginationFilters(req, &filters)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name           string
		filters        dto.PageFilter
		setupMock      func()
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Can get notifications with no pagination",
			setupMock: func() {
				registryMock.
					EXPECT().
					GetNotifications(gomock.Any(), dto.PageFilter{}).
					Return(dto.Page[dto.NotificationSummary]{
						Data:        []dto.NotificationSummary{},
						ResultCount: 0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Can get notifications with custom pagination",
			filters: dto.PageFilter{
				NextToken:  testutils.StrPtr("token"),
				MaxResults: testutils.IntPtr(20),
			},
			setupMock: func() {
				registryMock.
					EXPECT().
					GetNotifications(gomock.Any(), dto.PageFilter{
						NextToken:  testutils.StrPtr("token"),
						MaxResults: testutils.IntPtr(20),
					}).
					Return(dto.Page[dto.NotificationSummary]{
						Data:        []dto.NotificationSummary{},
						ResultCount: 0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Should fail if maxResults is less than 1",
			filters: dto.PageFilter{
				MaxResults: testutils.IntPtr(0),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'PageFilter.MaxResults' Error:Field validation for 'MaxResults' failed on the 'min' tag",
		},
		{
			name: "Should fail if there's an error retrieving notifications",
			setupMock: func() {
				registryMock.
					EXPECT().
					GetNotifications(gomock.Any(), gomock.Any()).
					Return(dto.Page[dto.NotificationSummary]{}, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			w := getNotifications(tt.filters)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], tt.expectedError)
			}
		})
	}
}
