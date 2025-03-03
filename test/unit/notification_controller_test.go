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
}

func testCreateNotification(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	mock.Publisher.
		EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	mock.Cache.
		EXPECT().
		UpdateNotificationStatus(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

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

	tests := []struct {
		name           string
		setupMock      func()
		modifyRequest  func(req dto.NotificationReq) dto.NotificationReq
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Can create new notifications",
			setupMock: func() {
				registryMock.EXPECT().
					SaveNotification(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(uuid.NewString(), nil)
			},
			modifyRequest:  func(req dto.NotificationReq) dto.NotificationReq { return req },
			expectedStatus: http.StatusNoContent,
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
				req.Title = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Title' Error:Field validation for 'Title' failed on the 'max' tag`,
		},
		{
			name: "Should fail if the title is empty",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.Title = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Title' Error:Field validation for 'Title' failed on the 'required' tag`,
		},
		{
			name: "Should fail if the contents are empty",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.Contents = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Contents' Error:Field validation for 'Contents' failed on the 'required' tag`,
		},
		{
			name: "Should fail if the contents exceeds the limits",
			modifyRequest: func(req dto.NotificationReq) dto.NotificationReq {
				req.Contents = testutils.MakeStrWithSize(1025)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  `Key: 'NotificationReq.Contents' Error:Field validation for 'Contents' failed on the 'max' tag`,
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

			req := testutils.MakeTestNotificationRequest()
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
	createdStatus := testutils.StatusPtr(controllers.Created)
	canceledStatus := testutils.StatusPtr(controllers.Canceled)

	expectedStatusLog := controllers.NotificationStatusLog{
		NotificationId: notificationId,
		Status:         *canceledStatus,
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
				sendingStatus := testutils.StatusPtr(controllers.Sending)
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
				sentStatus := testutils.StatusPtr(controllers.Sent)
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
				sendingStatus := testutils.StatusPtr(controllers.Sending)
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
				sentStatus := testutils.StatusPtr(controllers.Sent)
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
					Return(createdStatus, nil).
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
