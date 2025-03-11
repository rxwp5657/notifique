package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/notifique/internal"
	di "github.com/notifique/internal/di"
	"github.com/notifique/internal/dto"
	"github.com/notifique/internal/registry"
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
	testGetNotificationTemplateDetails(t, testApp.Engine, *testApp)
	testDeleteNotificationTemplate(t, testApp.Engine, *testApp)
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
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Name' Error:Field validation for 'Name' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the template name exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Name = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Name' Error:Field validation for 'Name' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the title template is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.TitleTemplate = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.TitleTemplate' Error:Field validation for 'TitleTemplate' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the title exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.TitleTemplate = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.TitleTemplate' Error:Field validation for 'TitleTemplate' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the contents template is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.ContentsTemplate = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.ContentsTemplate' Error:Field validation for 'ContentsTemplate' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the contents template exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.ContentsTemplate = testutils.MakeStrWithSize(4097)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.ContentsTemplate' Error:Field validation for 'ContentsTemplate' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the template description is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Description = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Description' Error:Field validation for 'Description' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the template description exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Description = testutils.MakeStrWithSize(257)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Description' Error:Field validation for 'Description' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the variable name is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Variables[0].Name = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Variables[0].Name' Error:Field validation for 'Name' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the variable name exceeds its max size",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Variables[0].Name = testutils.MakeStrWithSize(121)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Variables[0].Name' Error:Field validation for 'Name' failed on the 'max' tag`),
		},
		{
			name: "Should fail if the variable type is empty",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Variables[0].Type = ""
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Variables[0].Type' Error:Field validation for 'Type' failed on the 'required' tag`),
		},
		{
			name: "Should fail if the variable type is not a valid value",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				req.Variables[0].Type = "invalid"
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Variables[0].Type' Error:Field validation for 'Type' failed on the 'oneof' tag`),
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
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Variables' Error:Field validation for 'Variables' failed on the 'unique_var_name' tag`),
		},
		{
			name: "Should fail if the notification template has variables with a name that contains the separator",
			modifyRequest: func(req dto.NotificationTemplateReq) dto.NotificationTemplateReq {
				badName := dto.TemplateVariable{
					Name:     "bad~name",
					Type:     "STRING",
					Required: false,
				}
				req.Variables = append(req.Variables, badName)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateReq.Variables[2].Name' Error:Field validation for 'Name' failed on the 'templatevarname' tag`),
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
				GetTemplates(gomock.Any(), gomock.Any()).
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

func testGetNotificationTemplateDetails(t *testing.T, e *gin.Engine, mock di.MockedBackend) {

	userId := "1234"

	getTemplateDetails := func(params dto.NotificationTemplateUriParams) *httptest.ResponseRecorder {
		url := fmt.Sprintf("%s/%s", notificationsTemplateUrl, params.Id)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Add("userId", userId)
		e.ServeHTTP(w, req)
		return w
	}

	testReq := testutils.MakeTestNotificationTemplateRequest()

	testDetails := dto.NotificationTemplateDetails{
		Id:               uuid.NewString(),
		Name:             testReq.Name,
		Description:      testReq.Description,
		TitleTemplate:    testReq.TitleTemplate,
		ContentsTemplate: testReq.ContentsTemplate,
		Variables:        testReq.Variables,
		CreatedAt:        time.Now().Format(time.RFC3339),
		CreatedBy:        testUserId,
	}

	missingTemplateId := uuid.NewString()

	templateNotFound := fmt.Sprintf("Entity %v of type %v not found",
		missingTemplateId,
		registry.NotificationTemplateType,
	)

	tests := []struct {
		name           string
		setupMock      func()
		modifyParams   func(params dto.NotificationTemplateUriParams) dto.NotificationTemplateUriParams
		expectedStatus int
		expectedError  *string
		expectedResp   *dto.NotificationTemplateDetails
	}{
		{
			name:           "Can retrieve template details",
			expectedStatus: http.StatusOK,
			expectedResp:   &testDetails,
			modifyParams:   func(p dto.NotificationTemplateUriParams) dto.NotificationTemplateUriParams { return p },
			setupMock: func() {
				mock.Registry.MockNotificationTemplateRegistry.
					EXPECT().
					GetTemplateDetails(gomock.Any(), gomock.Any()).
					Return(testDetails, nil)
			},
		},
		{
			name: "Should fail if template id is not a valid UUID",
			modifyParams: func(p dto.NotificationTemplateUriParams) dto.NotificationTemplateUriParams {
				p.Id = "not-a-uuid"
				return p
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateUriParams.Id' Error:Field validation for 'Id' failed on the 'uuid' tag`),
		},
		{
			name: "Should fail if template doesn't exist",
			setupMock: func() {
				mock.Registry.MockNotificationTemplateRegistry.
					EXPECT().
					GetTemplateDetails(gomock.Any(), gomock.Any()).
					Return(dto.NotificationTemplateDetails{}, internal.EntityNotFound{
						Id:   missingTemplateId,
						Type: registry.NotificationTemplateType,
					})
			},
			modifyParams:   func(p dto.NotificationTemplateUriParams) dto.NotificationTemplateUriParams { return p },
			expectedStatus: http.StatusNotFound,
			expectedError:  testutils.StrPtr(templateNotFound),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			params := dto.NotificationTemplateUriParams{
				Id: uuid.NewString(),
			}

			w := getTemplateDetails(tt.modifyParams(params))

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != nil {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, resp["error"], *tt.expectedError)
			} else if tt.expectedResp != nil {
				resp := dto.NotificationTemplateDetails{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, *tt.expectedResp, resp)
			}
		})
	}
}

func testDeleteNotificationTemplate(t *testing.T, e *gin.Engine, mock di.MockedBackend) {

	userId := "1234"

	deleteTemplate := func(templateId string) *httptest.ResponseRecorder {
		url := fmt.Sprintf("%s/%s", notificationsTemplateUrl, templateId)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.Header.Add("userId", userId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name           string
		setupMock      func()
		modifyId       func(templateId string) string
		expectedStatus int
		expectedError  *string
	}{
		{
			name:           "should be able to delete a template",
			modifyId:       testutils.Echo[string],
			expectedStatus: http.StatusNoContent,
			setupMock: func() {
				mock.Registry.MockNotificationTemplateRegistry.
					EXPECT().
					DeleteTemplate(gomock.Any(), gomock.Any()).
					Return(nil)
			},
		},
		{
			name:           "should fail of the template id is not an uuid",
			modifyId:       func(templateId string) string { return "not an uuid" },
			expectedStatus: http.StatusBadRequest,
			expectedError:  testutils.StrPtr(`Key: 'NotificationTemplateUriParams.Id' Error:Field validation for 'Id' failed on the 'uuid' tag`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}

			w := deleteTemplate(tt.modifyId(uuid.NewString()))

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != nil {
				resp := make(map[string]string)
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal(err)
				}
				assert.Contains(t, *tt.expectedError, resp["error"])
			}
		})
	}
}
