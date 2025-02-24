package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	di "github.com/notifique/internal/di"
	"github.com/notifique/internal/server/dto"
	"github.com/notifique/internal/testutils"
)

const notificationsTemplateUrl = "/notifications/templates"

func TestNotificationTemplateController(t *testing.T) {

	controller := gomock.NewController(t)
	defer controller.Finish()

	testApp, err := di.InjectMockedBackend(context.TODO(), controller)

	if err != nil {
		t.Fatalf("failed to create mocked backend - %v", err)
	}

	testCreateNotificationTemplate(t, testApp.Engine, *testApp)
	testGetNotificationTemplates(t, testApp.Engine, *testApp)
}

func testCreateNotificationTemplate(t *testing.T, e *gin.Engine, mock di.MockedBackend) {

	userId := "1234"
	registryMock := mock.Registry.MockNotificationTemplateRegistry

	expectedResp := dto.NotificationTemplateCreatedResp{
		Id:        uuid.NewString(),
		Name:      "Test Template",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	createNotificationTemplate := func(templateReq dto.NotificationTemplateReq) *httptest.ResponseRecorder {
		body, _ := json.Marshal(templateReq)
		reader := bytes.NewReader(body)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, notificationsTemplateUrl, reader)
		req.Header.Add("userId", userId)
		e.ServeHTTP(w, req)
		return w
	}

	echoRequest := func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
		return req
	}

	strPtr := func(s string) *string {
		return &s
	}

	tests := []struct {
		name           string
		setupMock      func()
		modifyRequest  func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq
		expectedStatus int
		expectedError  *string
		expectedResp   *dto.NotificationTemplateCreatedResp
	}{
		{
			name: "Can create new notification template",
			setupMock: func() {
				registryMock.EXPECT().
					SaveTemplate(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(expectedResp, nil)
			},
			modifyRequest:  echoRequest,
			expectedStatus: http.StatusCreated,
			expectedResp:   &expectedResp,
		},
		{
			name: "Should fail if the template name is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Name = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Name' Error:Field validation for 'Name' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the template name exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Name = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Name' Error:Field validation for 'Name' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the title template is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.TitleTemplate = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.TitleTemplate' Error:Field validation for 'TitleTemplate' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the title exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.TitleTemplate = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.TitleTemplate' Error:Field validation for 'TitleTemplate' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the contents template is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.ContentsTemplate = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.ContentsTemplate' Error:Field validation for 'ContentsTemplate' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the contents template exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.ContentsTemplate = testutils.MakeStrWithSize(4097)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.ContentsTemplate' Error:Field validation for 'ContentsTemplate' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the template description is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Description = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Description' Error:Field validation for 'Description' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the template description exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Description = testutils.MakeStrWithSize(257)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Description' Error:Field validation for 'Description' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the variable name is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Variables[0].Name = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Variables[0].Name' Error:Field validation for 'Name' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the variable name exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Variables[0].Name = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Variables[0].Name' Error:Field validation for 'Name' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the variable type is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Variables[0].Type = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Variables[0].Type' Error:Field validation for 'Type' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the variable type is not a valid value",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Variables[0].Type = "invalid"
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Variables[0].Type' Error:Field validation for 'Type' failed on the 'oneof' tag`),
		},
		{
			name: "Should fail if the notification template has variables with the same name",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				duplicated := dto.TemplateVariable{
					Name:     "duplicated",
					Type:     "STRING",
					Required: false,
				}
				req.Variables = append(req.Variables, duplicated)
				req.Variables = append(req.Variables, duplicated)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  strPtr(`Key: 'NotificationTemplateReq.Variables' Error:Field validation for 'Variables' failed on the 'unique_var_name' tag`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			req := testutils.MakeTestNotificationTemplateRequest()
			req = tt.modifyRequest(req)
			w := createNotificationTemplate(req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != nil {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], *tt.expectedError)
			} else if tt.expectedResp != nil {
				resp := dto.NotificationTemplateCreatedResp{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, expectedResp, *tt.expectedResp)
			}
		})
	}
}

func testGetNotificationTemplates(t *testing.T, e *gin.Engine, mock di.MockedBackend) {

	userId := "1234"
	registryMock := mock.Registry.MockNotificationTemplateRegistry

	getNotificationTemplates := func(filters *dto.NotificationTemplateFilters) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, notificationsTemplateUrl, nil)
		testutils.AddNotificationTemplateFilters(req, filters)
		req.Header.Add("userId", userId)
		e.ServeHTTP(w, req)
		return w
	}

	pageSize := 3

	testPage := dto.Page[dto.NotificationTemplateInfoResp]{
		ResultCount: pageSize,
		Data:        testutils.MakeTestNotificationTemplateInfoResp(pageSize),
	}

	nilFilters := func(req dto.NotificationTemplateFilters) *dto.NotificationTemplateFilters {
		return nil
	}

	tests := []struct {
		name           string
		setupMock      func()
		modifyFilters  func(req dto.NotificationTemplateFilters) *dto.NotificationTemplateFilters
		expectedStatus int
		expectedError  *string
		expectedResp   *dto.Page[dto.NotificationTemplateInfoResp]
	}{{
		name:           "Can retrieve a page of notification templates",
		expectedStatus: http.StatusOK,
		expectedResp:   &testPage,
		modifyFilters:  nilFilters,
		setupMock: func() {
			registryMock.
				EXPECT().
				GetNotifications(gomock.Any(), gomock.Any()).
				Return(testPage, nil)
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			testFilters := testutils.MakeTestNotificationTemplateFilter()
			filters := tt.modifyFilters(testFilters)
			w := getNotificationTemplates(filters)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != nil {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], *tt.expectedError)
			} else if tt.expectedResp != nil {
				resp := dto.Page[dto.NotificationTemplateInfoResp]{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, resp, *tt.expectedResp)
			}
		})
	}
}
