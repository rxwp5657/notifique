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
	"github.com/notifique/service/internal"
	di "github.com/notifique/service/internal/di"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/service/internal/registry"
	"github.com/notifique/service/internal/testutils"
	"github.com/notifique/shared/auth"
	"github.com/notifique/shared/cache"
	sdto "github.com/notifique/shared/dto"
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
	testGetNotification(t, testApp.Engine, *testApp)
	testUpdateNotificationStatus(t, testApp.Engine, testApp)
	testGetNotificationStatus(t, testApp.Engine, testApp)
	testUpsertRecipientNotificationStatuses(t, testApp.Engine, testApp)
}

func testCreateNotification(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	mock.Publisher.
		EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	registryMock := mock.Registry.MockNotificationRegistry
	userId := "1234"

	createNotification := func(notificationReq sdto.NotificationReq) *httptest.ResponseRecorder {
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
		modifyRequest  func(req sdto.NotificationReq) sdto.NotificationReq
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Can create new notifications with raw contents",
			setupMock: func() {
				registryMock.EXPECT().
					SaveNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uuid.NewString(), nil)

				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)

				mock.Cache.
					EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(2)
			},
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.RawContents = &sdto.RawContents{
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
					Return([]sdto.TemplateVariable{
						{Name: "{user}", Type: "STRING", Required: true},
						{Name: "{date}", Type: "DATE", Required: true},
					}, nil)

				registryMock.EXPECT().
					SaveNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uuid.NewString(), nil)

				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)

				mock.Cache.
					EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					Times(2)
			},
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.TemplateContents = &sdto.TemplateContents{
					Id: uuid.NewString(),
					Variables: []sdto.TemplateVariableContents{
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
					Return(nil, internal.EntityNotFound{
						Id: randomTemplateId, Type: registry.NotificationTemplateType,
					})

				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)
			},
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.TemplateContents = &sdto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []sdto.TemplateVariableContents{{Name: "{user}", Value: "John"}},
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
					Return([]sdto.TemplateVariable{
						{Name: "{date}", Type: "DATE", Required: true},
					}, nil)

				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)
			},
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.TemplateContents = &sdto.TemplateContents{
					Id: uuid.NewString(),
					Variables: []sdto.TemplateVariableContents{{
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
					Return([]sdto.TemplateVariable{
						{Name: "{user}", Type: "STRING", Required: true},
					}, nil)

				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)
			},
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.TemplateContents = &sdto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []sdto.TemplateVariableContents{{Name: "{invalid}", Value: "value"}},
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
					Return([]sdto.TemplateVariable{
						{Name: "{user}", Type: "STRING", Required: true},
						{Name: "{date}", Type: "DATE", Required: true},
					}, nil)

				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)
			},
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.TemplateContents = &sdto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []sdto.TemplateVariableContents{{Name: "{user}", Value: "John"}},
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
					Return([]sdto.TemplateVariable{
						{Name: "{user}", Type: "STRING", Required: true, Validation: &pattern},
					}, nil)

				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)
			},
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.TemplateContents = &sdto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []sdto.TemplateVariableContents{{Name: "{user}", Value: "lowercase"}},
				}
				req.RawContents = nil
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "lowercase failed regex validation ^[A-Z]+$",
		},
		{
			name: "Should fail when neither raw contents nor template contents are provided",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.RawContents = nil
				req.TemplateContents = nil
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Error:Field validation for 'RawContents' failed on the 'required_without' tag",
		},
		{
			name: "Should fail when both raw contents and template contents are provided",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.RawContents = &sdto.RawContents{
					Title:    "Test Title",
					Contents: "Test Contents",
				}
				req.TemplateContents = &sdto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []sdto.TemplateVariableContents{{Name: "{user}", Value: "John"}},
				}
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Error:Field validation for 'RawContents' failed on the 'excluded_with' tag",
		},
		{
			name: "Should fail if the channel is not supported",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.Channels = append(req.Channels, "INVALID")
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `NotificationReq.Channels[2]' Error:Field validation for 'Channels[2]' failed on the 'oneof' tag`,
		},
		{
			name: "Should fail if the title exceeds the limits",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.RawContents.Title = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.RawContents.Title' Error:Field validation for 'Title' failed on the 'max' tag`,
		},
		{
			name: "Should fail if the title is empty",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.RawContents.Title = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.RawContents.Title' Error:Field validation for 'Title' failed on the 'required' tag`,
		},
		{
			name: "Should fail if the contents are empty",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.RawContents.Contents = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.RawContents.Contents' Error:Field validation for 'Contents' failed on the 'required' tag`,
		},
		{
			name: "Should fail if the contents exceeds the limits",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.RawContents.Contents = testutils.MakeStrWithSize(1025)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.RawContents.Contents' Error:Field validation for 'Contents' failed on the 'max' tag`,
		},
		{
			name: "Should fail if the topic exceeds the limits",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.Topic = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Topic' Error:Field validation for 'Topic' failed on the 'max' tag`,
		},
		{
			name: "Should fail if the topic is empty",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.Topic = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Topic' Error:Field validation for 'Topic' failed on the 'required' tag`,
		},
		{
			name: "Should fail if there are duplicated recipients",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.Recipients = append(req.Recipients, req.Recipients...)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Recipients' Error:Field validation for 'Recipients' failed on the 'unique' tag`,
		},
		{
			name: "Should fail if the priority is not supported",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.Priority = "Test"
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Priority' Error:Field validation for 'Priority' failed on the 'oneof' tag`,
		},
		{
			name: "Should fail if the image url is not an url",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				url := "not an url"
				req.Image = &url
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Image' Error:Field validation for 'Image' failed on the 'uri' tag`,
		},
		{
			name: "Should fail when a template name has the ~ separator character",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				req.RawContents = nil
				req.TemplateContents = &sdto.TemplateContents{
					Id:        uuid.NewString(),
					Variables: []sdto.TemplateVariableContents{{Name: "{u~ser}", Value: "John"}},
				}
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'NotificationReq.TemplateContents.Variables[0].Name' Error:Field validation for 'Name' failed on the 'templatevarname' tag",
		},
		{
			name: "Should fail if notification duplication check fails",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				return req
			},
			setupMock: func() {
				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", errors.New("cache error"), false)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Should return no content if notification already exists",
			modifyRequest: func(req sdto.NotificationReq) sdto.NotificationReq {
				return req
			},
			setupMock: func() {
				mock.Cache.
					EXPECT().Get(gomock.Any(), gomock.Any()).
					Return("", nil, true)
			},
			expectedStatus: http.StatusNoContent,
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
	createdStatus := sdto.Created
	canceledStatus := sdto.Canceled

	expectedStatusLog := sdto.NotificationStatusLog{
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
					Get(gomock.Any(), cache.GetNotificationStatusKey(notificationId)).
					Return(string(createdStatus), nil, true).
					Times(1)

				mock.Cache.EXPECT().
					Set(gomock.Any(),
						cache.GetNotificationStatusKey(expectedStatusLog.NotificationId),
						string(expectedStatusLog.Status),
						gomock.Any()).
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
					Get(gomock.Any(), cache.GetNotificationStatusKey(notificationId)).
					Return("", nil, false).
					Times(1)

				mock.Registry.MockNotificationRegistry.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(createdStatus, nil).
					Times(1)

				mock.Cache.EXPECT().
					Set(gomock.Any(),
						cache.GetNotificationStatusKey(expectedStatusLog.NotificationId),
						string(expectedStatusLog.Status),
						gomock.Any()).
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
				mock.Cache.
					EXPECT().
					Get(gomock.Any(), cache.GetNotificationStatusKey(notificationId)).
					Return(string(sdto.Sending), nil, true).
					Times(1)
			},
		},
		{
			name:           "Should fail if the status of the notification is SENT (status from cache)",
			notificationId: notificationId,
			expectedStatus: 400,
			expectedError:  "Notification has been sent",
			setupMock: func() {
				mock.Cache.
					EXPECT().
					Get(gomock.Any(), cache.GetNotificationStatusKey(notificationId)).
					Return(string(sdto.Sent), nil, true).
					Times(1)
			},
		},
		{
			name:           "Should fail if the status of the notification is SENDING (status from db)",
			notificationId: notificationId,
			expectedStatus: 400,
			expectedError:  "Notification is being sent",
			setupMock: func() {
				sendingStatus := sdto.Sending
				mock.Cache.
					EXPECT().
					Get(gomock.Any(), cache.GetNotificationStatusKey(notificationId)).
					Return("", nil, false).
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
				sentStatus := sdto.Sent
				mock.Cache.
					EXPECT().
					Get(gomock.Any(), cache.GetNotificationStatusKey(notificationId)).
					Return("", nil, false).
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
					Get(gomock.Any(), cache.GetNotificationStatusKey(notificationId)).
					Return(string(sdto.Created), nil, true).
					Times(1)

				mock.Cache.EXPECT().
					Set(gomock.Any(),
						cache.GetNotificationStatusKey(expectedStatusLog.NotificationId),
						string(expectedStatusLog.Status),
						gomock.Any()).
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
	getNotifications := func(filters sdto.PageFilter) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, notificationsUrl, nil)
		req.Header.Add("userId", userId)
		testutils.AddPaginationFilters(req, &filters)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name           string
		filters        sdto.PageFilter
		setupMock      func()
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Can get notifications with no pagination",
			setupMock: func() {
				registryMock.
					EXPECT().
					GetNotifications(gomock.Any(), sdto.PageFilter{}).
					Return(sdto.Page[dto.NotificationSummary]{
						Data:        []dto.NotificationSummary{},
						ResultCount: 0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Can get notifications with custom pagination",
			filters: sdto.PageFilter{
				NextToken:  testutils.StrPtr("token"),
				MaxResults: testutils.IntPtr(20),
			},
			setupMock: func() {
				registryMock.
					EXPECT().
					GetNotifications(gomock.Any(), sdto.PageFilter{
						NextToken:  testutils.StrPtr("token"),
						MaxResults: testutils.IntPtr(20),
					}).
					Return(sdto.Page[dto.NotificationSummary]{
						Data:        []dto.NotificationSummary{},
						ResultCount: 0,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Should fail if maxResults is less than 1",
			filters: sdto.PageFilter{
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
					Return(sdto.Page[dto.NotificationSummary]{}, errors.New("db error"))
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

func testGetNotification(t *testing.T, e *gin.Engine, mock di.MockedBackend) {

	userId := "1234"
	registryMock := mock.Registry.MockNotificationRegistry

	getNotification := func(notificationId string) *httptest.ResponseRecorder {
		url := fmt.Sprintf("%s/%s", notificationsUrl, notificationId)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, url, nil)
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
			name:           "Can get a notification",
			notificationId: uuid.NewString(),
			expectedStatus: http.StatusOK,
			setupMock: func() {
				registryMock.
					EXPECT().
					GetNotification(gomock.Any(), gomock.Any()).
					Return(dto.NotificationResp{}, nil)
			},
		},
		{
			name:           "Should fail when notification doesn't exist",
			notificationId: uuid.NewString(),
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
			setupMock: func() {
				registryMock.
					EXPECT().
					GetNotification(gomock.Any(), gomock.Any()).
					Return(dto.NotificationResp{}, internal.EntityNotFound{})
			},
		},
		{
			name:           "Should fail if the id is not an uuid",
			notificationId: "not-an-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Key: 'NotificationUriParams.NotificationId' Error:Field validation for 'NotificationId' failed on the 'uuid' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			w := getNotification(tt.notificationId)

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

func testUpdateNotificationStatus(t *testing.T, e *gin.Engine, mock *di.MockedBackend) {
	notificationId := uuid.NewString()
	statusUrl := fmt.Sprintf("/notifications/%s/status", notificationId)

	updateStatus := func(status sdto.NotificationStatusLog) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		marshalled, _ := json.Marshal(status)
		reader := bytes.NewReader(marshalled)
		req, _ := http.NewRequest(http.MethodPatch, statusUrl, reader)
		req.Header.Add(string(auth.UserHeader), testUserId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name          string
		status        sdto.NotificationStatusLog
		expectedCode  int
		expectedError string
		mockSetup     func()
	}{
		{
			name: "Should update notification status",
			status: sdto.NotificationStatusLog{
				NotificationId: notificationId,
				Status:         sdto.Sending,
			},
			expectedCode: http.StatusNoContent,
			mockSetup: func() {
				mock.Registry.MockNotificationRegistry.
					EXPECT().
					UpdateNotificationStatus(gomock.Any(), sdto.NotificationStatusLog{
						NotificationId: notificationId,
						Status:         sdto.Sending,
					}).
					Return(nil)

				mock.Cache.
					EXPECT().
					Set(gomock.Any(), gomock.Any(), string(sdto.Sending), gomock.Any()).
					Return(nil)
			},
		},
		{
			name: "Should fail to update with invalid status",
			status: sdto.NotificationStatusLog{
				NotificationId: notificationId,
				Status:         "INVALID",
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Field validation for 'Status' failed on the 'oneof' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			w := updateStatus(tt.status)
			assert.Equal(t, tt.expectedCode, w.Code)

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

func testGetNotificationStatus(t *testing.T, e *gin.Engine, mock *di.MockedBackend) {
	notificationId := uuid.NewString()
	statusUrl := fmt.Sprintf("/notifications/%s/status", notificationId)

	getStatus := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, statusUrl, nil)
		req.Header.Add(string(auth.UserHeader), testUserId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name           string
		expectedCode   int
		expectedError  string
		expectedStatus sdto.NotificationStatus
		mockSetup      func()
	}{
		{
			name:           "Should get notification status from cache",
			expectedCode:   http.StatusOK,
			expectedStatus: sdto.Sending,
			mockSetup: func() {
				mock.Cache.
					EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(string(sdto.Sending), nil, true)
			},
		},
		{
			name:           "Should get notification status from registry when not in cache",
			expectedCode:   http.StatusOK,
			expectedStatus: sdto.Sending,
			mockSetup: func() {
				mock.Cache.
					EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)

				mock.Registry.MockNotificationRegistry.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(sdto.Sending, nil)
			},
		},
		{
			name:          "Should return 404 when notification not found",
			expectedCode:  http.StatusNotFound,
			expectedError: "Notification not found",
			mockSetup: func() {
				mock.Cache.
					EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return("", nil, false)

				mock.Registry.MockNotificationRegistry.
					EXPECT().
					GetNotificationStatus(gomock.Any(), notificationId).
					Return(sdto.NotificationStatus(""), internal.EntityNotFound{
						Id:   notificationId,
						Type: registry.NotificationType,
					})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup()
			}

			w := getStatus()
			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedError != "" {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], tt.expectedError)
			} else {
				var resp sdto.NotificationStatusResp
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tt.expectedStatus, resp.Status)
			}
		})
	}
}

func testUpsertRecipientNotificationStatuses(t *testing.T, e *gin.Engine, mock *di.MockedBackend) {

	notificationId := uuid.NewString()
	url := fmt.Sprintf("/notifications/%s/recipients/statuses", notificationId)

	upsertStatuses := func(statuses []sdto.RecipientNotificationStatus) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		body, _ := json.Marshal(statuses)
		reader := bytes.NewReader(body)
		req, _ := http.NewRequest(http.MethodPost, url, reader)
		req.Header.Add("userId", testUserId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name           string
		statuses       []sdto.RecipientNotificationStatus
		setupMock      func()
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Can upsert recipient notification statuses",
			statuses: []sdto.RecipientNotificationStatus{
				{
					UserId:  "user1",
					Channel: "e-mail",
					Status:  "SENT",
				},
			},
			setupMock: func() {
				mock.Registry.MockNotificationRegistry.
					EXPECT().
					UpsertRecipientNotificationStatuses(
						gomock.Any(),
						notificationId,
						gomock.Any(),
					).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "Should fail when notification doesn't exist",
			statuses: []sdto.RecipientNotificationStatus{
				{
					UserId:  "user1",
					Channel: "e-mail",
					Status:  "SENT",
				},
			},
			setupMock: func() {
				mock.Registry.MockNotificationRegistry.
					EXPECT().
					UpsertRecipientNotificationStatuses(
						gomock.Any(),
						notificationId,
						gomock.Any(),
					).Return(internal.EntityNotFound{})
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
		},
		{
			name: "Should fail with invalid channel",
			statuses: []sdto.RecipientNotificationStatus{
				{
					UserId:  "user1",
					Channel: "invalid",
					Status:  "SENT",
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Field validation for 'Channel' failed on the 'oneof' tag",
		},
		{
			name: "Should fail with invalid status",
			statuses: []sdto.RecipientNotificationStatus{
				{
					UserId:  "user1",
					Channel: "e-mail",
					Status:  "INVALID",
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Field validation for 'Status' failed on the 'oneof' tag",
		},
		{
			name: "Should fail with empty userId",
			statuses: []sdto.RecipientNotificationStatus{
				{
					UserId:  "",
					Channel: "e-mail",
					Status:  "SENT",
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Field validation for 'UserId' failed on the 'required' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			w := upsertStatuses(tt.statuses)
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
