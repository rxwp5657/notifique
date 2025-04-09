package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/notifique/service/internal"
	di "github.com/notifique/service/internal/di"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/service/internal/registry"
	"github.com/notifique/service/internal/testutils"
	"github.com/notifique/shared/auth"
	"github.com/notifique/shared/cache"
	sdto "github.com/notifique/shared/dto"
)

const distributionListUrl = "/distribution-lists"
const testUserId = "1234"
const distributionListKey = "notifications:endpoint:2249993f9e59254124395cab5dfac567:/distribution-lists*"
const distributionListRecipientsKey = "notifications:endpoint:e313a18491a6adbebd6d0a2bc056ee71:/distribution-lists/Test/recipients*"

func TestDistributionListController(t *testing.T) {

	controller := gomock.NewController(t)
	defer controller.Finish()

	testApp, err := di.InjectMockedBackend(context.TODO(), controller)

	if err != nil {
		t.Fatalf("failed to create mocked backend - %v", err)
	}

	testCreateDistributionList(t, testApp.Engine, *testApp)
	testAddRecipients(t, testApp.Engine, *testApp)
	testDeleteRecipients(t, testApp.Engine, *testApp)
	testDeleteDistributionList(t, testApp.Engine, *testApp)
	testGetDistributionLists(t, testApp.Engine, *testApp)
	testGetDistributionListRescipients(t, testApp.Engine, *testApp)
}

func testCreateDistributionList(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	createDistributionList := func(dl dto.DistributionList) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		marshalled, _ := json.Marshal(dl)
		reader := bytes.NewReader(marshalled)
		req, _ := http.NewRequest(http.MethodPost, distributionListUrl, reader)
		req.Header.Add(string(auth.UserHeader), testUserId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name          string
		input         dto.DistributionList
		setupMock     func()
		expectedCode  int
		expectedError string
	}{
		{
			name:  "Success - Create distribution list",
			input: dl,
			setupMock: func() {
				mock.Registry.
					MockDistributionRegistry.
					EXPECT().
					CreateDistributionList(gomock.Any(), gomock.Any()).Return(nil)

				mock.Cache.
					EXPECT().
					DelWithPrefix(gomock.Any(), cache.Key(distributionListKey)).
					Return(nil)
			},
			expectedCode: http.StatusCreated,
		},
		{
			name:  "Fail - Distribution list already exists",
			input: dl,
			setupMock: func() {
				mock.Registry.
					MockDistributionRegistry.
					EXPECT().
					CreateDistributionList(gomock.Any(), gomock.Any()).
					Return(internal.DistributionListAlreadyExists{Name: dl.Name})
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: fmt.Sprintf("distribution list %s already exists", dl.Name),
		},
		{
			name:          "Fail - Name too short",
			input:         dto.DistributionList{Name: "A", Recipients: []string{}},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "DistributionList.Name' Error:Field validation for 'Name' failed on the 'min' tag",
		},
		{
			name:          "Fail - Name too long",
			input:         dto.DistributionList{Name: strings.Repeat("a", 121), Recipients: []string{}},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "DistributionList.Name' Error:Field validation for 'Name' failed on the 'max' tag",
		},
		{
			name:          "Fail - Invalid name format",
			input:         dto.DistributionList{Name: "Test/Invalid", Recipients: []string{}},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "DistributionList.Name' Error:Field validation for 'Name' failed on the 'distributionlistname' tag",
		},
		{
			name:          "Fail - Too many recipients",
			input:         dto.DistributionList{Name: "Test", Recipients: testutils.MakeRecipients(257)},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "DistributionList.Recipients' Error:Field validation for 'Recipients' failed on the 'max' tag",
		},
		{
			name:          "Fail - Duplicate recipients",
			input:         dto.DistributionList{Name: "Test", Recipients: []string{"1", "1"}},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "DistributionList.Recipients' Error:Field validation for 'Recipients' failed on the 'unique' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			w := createDistributionList(tt.input)
			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedError != "" {
				resp := make(map[string]string)
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Contains(t, resp["error"], tt.expectedError)
			}
		})
	}
}

func testAddRecipients(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	addRecipients := func(dlName string, recipients []string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		body := map[string][]string{"recipients": recipients}
		marshalled, _ := json.Marshal(body)
		reader := bytes.NewReader(marshalled)
		url := fmt.Sprintf("%s/%s/recipients", distributionListUrl, dlName)
		req, _ := http.NewRequest(http.MethodPatch, url, reader)
		req.Header.Add(string(auth.UserHeader), testUserId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name          string
		dlName        string
		recipients    []string
		setupMock     func()
		expectedCode  int
		expectedError string
		expectedResp  *dto.DistributionListSummary
	}{
		{
			name:       "Success - Add recipients",
			dlName:     dl.Name,
			recipients: []string{"4", "5", "6"},
			setupMock: func() {
				mock.Registry.MockDistributionRegistry.
					EXPECT().
					AddRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&dto.DistributionListSummary{
						Name:               dl.Name,
						NumberOfRecipients: 6,
					}, nil)

				mock.Cache.
					EXPECT().
					DelWithPrefix(gomock.Any(), cache.Key(distributionListRecipientsKey)).
					Return(nil)
			},
			expectedCode: http.StatusOK,
			expectedResp: &dto.DistributionListSummary{
				Name:               dl.Name,
				NumberOfRecipients: 6,
			},
		},
		{
			name:       "Fail - Distribution list not found",
			dlName:     dl.Name,
			recipients: []string{"4", "5", "6"},
			setupMock: func() {
				mock.Registry.MockDistributionRegistry.
					EXPECT().
					AddRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, internal.EntityNotFound{
						Id:   dl.Name,
						Type: registry.DistributionListType,
					})
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: fmt.Sprintf("entity %v of type %v not found", dl.Name, registry.DistributionListType),
		},
		{
			name:          "Fail - Empty recipients list",
			dlName:        dl.Name,
			recipients:    []string{},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Error:Field validation for 'Recipients' failed on the 'min' tag",
		},
		{
			name:          "Fail - Too many recipients",
			dlName:        dl.Name,
			recipients:    testutils.MakeRecipients(257),
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Error:Field validation for 'Recipients' failed on the 'max' tag",
		},
		{
			name:          "Fail - Empty user id",
			dlName:        dl.Name,
			recipients:    []string{""},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Error:Field validation for 'Recipients[0]' failed on the 'min' tag",
		},
		{
			name:          "Fail - Duplicate recipients",
			dlName:        dl.Name,
			recipients:    []string{"1", "1"},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Error:Field validation for 'Recipients' failed on the 'unique' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			w := addRecipients(tt.dlName, tt.recipients)
			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedError != "" {
				resp := make(map[string]string)
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Contains(t, resp["error"], tt.expectedError)
			}

			if tt.expectedResp != nil {
				var resp dto.DistributionListSummary
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Equal(t, tt.expectedResp, &resp)
			}
		})
	}
}

func testDeleteRecipients(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	deleteRecipients := func(dlName string, recipients []string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		body := map[string][]string{"recipients": recipients}
		marshalled, _ := json.Marshal(body)
		reader := bytes.NewReader(marshalled)
		url := fmt.Sprintf("%s/%s/recipients", distributionListUrl, dlName)
		req, _ := http.NewRequest(http.MethodDelete, url, reader)
		req.Header.Add(string(auth.UserHeader), testUserId)
		e.ServeHTTP(w, req)
		return w
	}

	tests := []struct {
		name          string
		dlName        string
		recipients    []string
		setupMock     func()
		expectedCode  int
		expectedError string
		expectedResp  *dto.DistributionListSummary
	}{
		{
			name:       "Success - Delete recipients",
			dlName:     dl.Name,
			recipients: []string{"1", "2"},
			setupMock: func() {
				mock.Registry.MockDistributionRegistry.
					EXPECT().
					DeleteRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&dto.DistributionListSummary{
						Name:               dl.Name,
						NumberOfRecipients: 1,
					}, nil)

				mock.Cache.
					EXPECT().
					DelWithPrefix(gomock.Any(), cache.Key(distributionListRecipientsKey)).
					Return(nil)
			},
			expectedCode: http.StatusOK,
			expectedResp: &dto.DistributionListSummary{
				Name:               dl.Name,
				NumberOfRecipients: 1,
			},
		},
		{
			name:       "Fail - Distribution list not found",
			dlName:     dl.Name,
			recipients: []string{"1", "2"},
			setupMock: func() {
				mock.Registry.MockDistributionRegistry.
					EXPECT().
					DeleteRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, internal.EntityNotFound{
						Id:   dl.Name,
						Type: registry.DistributionListType,
					})
			},
			expectedCode:  http.StatusBadRequest,
			expectedError: fmt.Sprintf("entity %v of type %v not found", dl.Name, registry.DistributionListType),
		},
		{
			name:          "Fail - Empty recipients list",
			dlName:        dl.Name,
			recipients:    []string{},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Error:Field validation for 'Recipients' failed on the 'min' tag",
		},
		{
			name:          "Fail - Too many recipients",
			dlName:        dl.Name,
			recipients:    testutils.MakeRecipients(257),
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Error:Field validation for 'Recipients' failed on the 'max' tag",
		},
		{
			name:          "Fail - Empty user id",
			dlName:        dl.Name,
			recipients:    []string{""},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Error:Field validation for 'Recipients[0]' failed on the 'min' tag",
		},
		{
			name:          "Fail - Duplicate recipients",
			dlName:        dl.Name,
			recipients:    []string{"1", "1"},
			setupMock:     func() {},
			expectedCode:  http.StatusBadRequest,
			expectedError: "Error:Field validation for 'Recipients' failed on the 'unique' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()
			w := deleteRecipients(tt.dlName, tt.recipients)
			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedError != "" {
				resp := make(map[string]string)
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Contains(t, resp["error"], tt.expectedError)
			}

			if tt.expectedResp != nil {
				var resp dto.DistributionListSummary
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Equal(t, tt.expectedResp, &resp)
			}
		})
	}
}

func testDeleteDistributionList(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	testDL := "Test"

	deleteDistributionList := func(dlName string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		url := fmt.Sprintf("%s/%s", distributionListUrl, dlName)

		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.Header.Add(string(auth.UserHeader), testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to delete a distribution list", func(t *testing.T) {
		mock.Registry.MockDistributionRegistry.
			EXPECT().
			DeleteDistributionList(gomock.Any(), gomock.Any()).
			Return(nil)

		mock.Cache.
			EXPECT().
			DelWithPrefix(gomock.Any(), cache.Key(distributionListRecipientsKey)).
			Return(nil)

		mock.Cache.
			EXPECT().
			DelWithPrefix(gomock.Any(), cache.Key(distributionListKey)).
			Return(nil)

		w := deleteDistributionList(testDL)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func testGetDistributionLists(t *testing.T, e *gin.Engine, mock di.MockedBackend) {
	numLists := 3
	summaries := testutils.MakeSummaries(testutils.MakeDistributionLists(numLists))
	page := sdto.Page[dto.DistributionListSummary]{
		NextToken:   nil,
		PrevToken:   nil,
		ResultCount: len(summaries),
		Data:        summaries,
	}

	getDistributionLists := func() *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodGet, distributionListUrl, nil)
		req.Header.Add(string(auth.UserHeader), testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to run retrieve the first page of distribution lists", func(t *testing.T) {
		mock.Registry.MockDistributionRegistry.
			EXPECT().
			GetDistributionLists(gomock.Any(), gomock.Any()).
			Return(page, nil)

		w := getDistributionLists()

		resp := sdto.Page[dto.DistributionListSummary]{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, w.Code, http.StatusOK)
		assert.Equal(t, page, resp)
	})
}

func testGetDistributionListRescipients(t *testing.T, e *gin.Engine, mock di.MockedBackend) {

	dlName := "Test"
	recipients := testutils.MakeRecipients(10)

	recipientsPage := sdto.Page[string]{
		NextToken:   nil,
		PrevToken:   nil,
		ResultCount: len(recipients),
		Data:        recipients,
	}

	getRecipients := func(dlName string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		url := fmt.Sprintf("%s/%s/recipients", distributionListUrl, dlName)

		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Add(string(auth.UserHeader), testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to retrieve the recipients of a distribution list", func(t *testing.T) {
		mock.Registry.MockDistributionRegistry.
			EXPECT().
			GetRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(recipientsPage, nil)

		w := getRecipients(dlName)

		resp := sdto.Page[string]{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, recipientsPage, resp)
	})

	t.Run("Should return 404 if the distribution list doesn't exists", func(t *testing.T) {
		mock.Registry.MockDistributionRegistry.
			EXPECT().
			GetRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(sdto.Page[string]{}, internal.EntityNotFound{
				Id:   dlName,
				Type: registry.DistributionListType,
			})

		w := getRecipients(dlName)

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := fmt.Sprintf("entity %v of type %v not found",
			dlName,
			registry.DistributionListType,
		)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})
}
