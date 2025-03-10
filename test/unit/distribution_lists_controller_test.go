package unit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	di "github.com/notifique/internal/di"
	"github.com/notifique/internal/registry"
	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/dto"
	"github.com/notifique/internal/testutils"
	mk "github.com/notifique/internal/testutils/mocks"
)

const distributionListUrl = "/distribution-lists"
const testUserId = "1234"

func TestDistributionListController(t *testing.T) {

	controller := gomock.NewController(t)
	defer controller.Finish()

	testApp, err := di.InjectMockedBackend(context.TODO(), controller)

	if err != nil {
		t.Fatalf("failed to create mocked backend - %v", err)
	}

	testCreateDistributionList(t, testApp.Engine, *testApp.Registry.MockDistributionRegistry)
	testAddRecipients(t, testApp.Engine, *testApp.Registry.MockDistributionRegistry)
	testDeleteRecipients(t, testApp.Engine, *testApp.Registry.MockDistributionRegistry)
	testDeleteDistributionList(t, testApp.Engine, *testApp.Registry.MockDistributionRegistry)
	testGetDistributionLists(t, testApp.Engine, *testApp.Registry.MockDistributionRegistry)
	testGetDistributionListRescipients(t, testApp.Engine, *testApp.Registry.MockDistributionRegistry)
}

func testCreateDistributionList(t *testing.T, e *gin.Engine, mock mk.MockDistributionRegistry) {

	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	createDistributionList := func(dl dto.DistributionList) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()

		marshalled, _ := json.Marshal(dl)
		reader := bytes.NewReader(marshalled)

		req, _ := http.NewRequest(http.MethodPost, distributionListUrl, reader)
		req.Header.Add("userId", testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to create a distribution list", func(t *testing.T) {

		mock.
			EXPECT().
			CreateDistributionList(gomock.Any(), gomock.Any()).
			Return(nil)

		w := createDistributionList(dl)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Should fail if the distribution list already exists", func(t *testing.T) {

		mock.
			EXPECT().
			CreateDistributionList(gomock.Any(), gomock.Any()).
			Return(server.DistributionListAlreadyExists{
				Name: dl.Name,
			})

		w := createDistributionList(dl)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		errTemplate := "Distribution list %s already exists"
		expectedMsg := fmt.Sprintf(errTemplate, dl.Name)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, expectedMsg, resp["error"])
	})

	t.Run("Should fail if the distribution list name is less than the allowed min", func(t *testing.T) {
		dl := dto.DistributionList{
			Name:       "A",
			Recipients: []string{},
		}

		w := createDistributionList(dl)

		expectedMsg := "DistributionList.Name' Error:Field validation for 'Name' failed on the 'min' tag"

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the distribution list name is greather than the allowed max", func(t *testing.T) {

		maxNameLength := 120

		name := ""

		for range maxNameLength + 1 {
			name += "a"
		}

		dl := dto.DistributionList{
			Name:       name,
			Recipients: []string{},
		}

		w := createDistributionList(dl)

		expectedMsg := "DistributionList.Name' Error:Field validation for 'Name' failed on the 'max' tag"

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the distribution list name doesn't comply whith the name spec", func(t *testing.T) {
		dl := dto.DistributionList{
			Name:       "TestDL/BadToken",
			Recipients: []string{},
		}

		w := createDistributionList(dl)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "DistributionList.Name' Error:Field validation for 'Name' failed on the 'distributionlistname' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the distribution list name already exists", func(t *testing.T) {
		mock.EXPECT().
			CreateDistributionList(gomock.Any(), gomock.Any()).
			Return(server.DistributionListAlreadyExists{Name: dl.Name})

		w := createDistributionList(dl)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := fmt.Sprintf("Distribution list %s already exists", dl.Name)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if the number of recipients is greather than the maximum allowed", func(t *testing.T) {

		dl := dto.DistributionList{
			Name:       "Test",
			Recipients: testutils.MakeRecipients(257),
		}

		w := createDistributionList(dl)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "DistributionList.Recipients' Error:Field validation for 'Recipients' failed on the 'max' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail if there are duplicated recipients", func(t *testing.T) {
		recipients := []string{"1", "1"}

		dl := dto.DistributionList{
			Name:       "Test",
			Recipients: recipients,
		}

		w := createDistributionList(dl)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "DistributionList.Recipients' Error:Field validation for 'Recipients' failed on the 'unique' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})
}

func testAddRecipients(t *testing.T, e *gin.Engine, mock mk.MockDistributionRegistry) {

	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	newRecipients := []string{"4", "5", "6"}

	addRecipients := func(dlName string, recipients []string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		body := make(map[string][]string)
		body["recipients"] = recipients

		marshalled, _ := json.Marshal(body)
		reader := bytes.NewReader(marshalled)

		url := fmt.Sprintf("%s/%s/recipients", distributionListUrl, dlName)

		req, _ := http.NewRequest(http.MethodPatch, url, reader)
		req.Header.Add("userId", testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Can add recipients", func(t *testing.T) {

		mock.
			EXPECT().
			AddRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&dto.DistributionListSummary{
				Name:               dl.Name,
				NumberOfRecipients: len(dl.Recipients) + len(newRecipients),
			}, nil)

		w := addRecipients(dl.Name, newRecipients)

		resp := dto.DistributionListSummary{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, resp.Name, dl.Name)
		assert.Equal(t, resp.NumberOfRecipients, len(dl.Recipients)+len(newRecipients))
	})

	t.Run("Should fail if the distribution list doesn't exist", func(t *testing.T) {
		mock.
			EXPECT().
			AddRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, server.EntityNotFound{
				Id:   dl.Name,
				Type: registry.DistributionListType,
			})

		w := addRecipients(dl.Name, newRecipients)

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := fmt.Sprintf("Entity %v of type %v not found",
			dl.Name,
			registry.DistributionListType,
		)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, expectedMsg, resp["error"])
	})

	t.Run("Should fail when sending an empty list of recipients", func(t *testing.T) {

		w := addRecipients(dl.Name, []string{})

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Error:Field validation for 'Recipients' failed on the 'min' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail when exceeding the number of recipients", func(t *testing.T) {
		w := addRecipients(dl.Name, testutils.MakeRecipients(257))

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Error:Field validation for 'Recipients' failed on the 'max' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail when trying to add an empty user id", func(t *testing.T) {
		w := addRecipients(dl.Name, []string{""})

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Error:Field validation for 'Recipients[0]' failed on the 'min' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail when supplying duplicated user ids", func(t *testing.T) {
		w := addRecipients(dl.Name, []string{"1", "1"})

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Error:Field validation for 'Recipients' failed on the 'unique' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})
}

func testDeleteRecipients(t *testing.T, e *gin.Engine, mock mk.MockDistributionRegistry) {

	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	recipientsToDelete := []string{"1", "2"}

	deleteRecipients := func(dlName string, recipients []string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		body := make(map[string][]string)
		body["recipients"] = recipients

		marshalled, _ := json.Marshal(body)
		reader := bytes.NewReader(marshalled)

		url := fmt.Sprintf("%s/%s/recipients", distributionListUrl, dlName)

		req, _ := http.NewRequest(http.MethodDelete, url, reader)
		req.Header.Add("userId", testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Can delete recipients", func(t *testing.T) {

		mock.
			EXPECT().
			DeleteRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&dto.DistributionListSummary{
				Name:               dl.Name,
				NumberOfRecipients: 1,
			}, nil)

		w := deleteRecipients(dl.Name, recipientsToDelete)

		resp := dto.DistributionListSummary{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, resp.Name, dl.Name)
		assert.Equal(t, resp.NumberOfRecipients, 1)
	})

	t.Run("Should fail if the distribution list doesn't exist", func(t *testing.T) {
		mock.
			EXPECT().
			DeleteRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, server.EntityNotFound{
				Id:   dl.Name,
				Type: registry.DistributionListType,
			})

		w := deleteRecipients(dl.Name, recipientsToDelete)

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := fmt.Sprintf("Entity %v of type %v not found",
			dl.Name,
			registry.DistributionListType,
		)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, expectedMsg, resp["error"])
	})

	t.Run("Should fail when sending an empty list of recipients", func(t *testing.T) {

		w := deleteRecipients(dl.Name, []string{})

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Error:Field validation for 'Recipients' failed on the 'min' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail when exceeding the number of recipients", func(t *testing.T) {

		w := deleteRecipients(dl.Name, testutils.MakeRecipients(257))

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Error:Field validation for 'Recipients' failed on the 'max' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail when trying to add an empty user id", func(t *testing.T) {
		w := deleteRecipients(dl.Name, []string{""})

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Error:Field validation for 'Recipients[0]' failed on the 'min' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})

	t.Run("Should fail when supplying duplicated user ids", func(t *testing.T) {
		w := deleteRecipients(dl.Name, []string{"1", "1"})

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := "Error:Field validation for 'Recipients' failed on the 'unique' tag"

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})
}

func testDeleteDistributionList(t *testing.T, e *gin.Engine, mock mk.MockDistributionRegistry) {
	testDL := "Test"

	deleteDistributionList := func(dlName string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		url := fmt.Sprintf("%s/%s", distributionListUrl, dlName)

		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		req.Header.Add("userId", testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to delete a distribution list", func(t *testing.T) {
		mock.EXPECT().
			DeleteDistributionList(gomock.Any(), gomock.Any()).
			Return(nil)

		w := deleteDistributionList(testDL)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func testGetDistributionLists(t *testing.T, e *gin.Engine, mock mk.MockDistributionRegistry) {
	numLists := 3
	summaries := testutils.MakeSummaries(testutils.MakeDistributionLists(numLists))
	page := dto.Page[dto.DistributionListSummary]{
		NextToken:   nil,
		PrevToken:   nil,
		ResultCount: len(summaries),
		Data:        summaries,
	}

	getDistributionLists := func() *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		req, _ := http.NewRequest(http.MethodGet, distributionListUrl, nil)
		req.Header.Add("userId", testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to run retrieve the first page of distribution lists", func(t *testing.T) {
		mock.EXPECT().
			GetDistributionLists(gomock.Any(), gomock.Any()).
			Return(page, nil)

		w := getDistributionLists()

		resp := dto.Page[dto.DistributionListSummary]{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, w.Code, http.StatusOK)
		assert.Equal(t, page, resp)
	})
}

func testGetDistributionListRescipients(t *testing.T, e *gin.Engine, mock mk.MockDistributionRegistry) {

	dlName := "Test"
	recipients := testutils.MakeRecipients(10)

	recipientsPage := dto.Page[string]{
		NextToken:   nil,
		PrevToken:   nil,
		ResultCount: len(recipients),
		Data:        recipients,
	}

	getRecipients := func(dlName string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		url := fmt.Sprintf("%s/%s/recipients", distributionListUrl, dlName)

		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Add("userId", testUserId)

		e.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to retrieve the recipients of a distribution list", func(t *testing.T) {
		mock.EXPECT().
			GetRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(recipientsPage, nil)

		w := getRecipients(dlName)

		resp := dto.Page[string]{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, recipientsPage, resp)
	})

	t.Run("Should return 404 if the distribution list doesn't exists", func(t *testing.T) {
		mock.EXPECT().
			GetRecipients(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(dto.Page[string]{}, server.EntityNotFound{
				Id:   dlName,
				Type: registry.DistributionListType,
			})

		w := getRecipients(dlName)

		resp := map[string]string{}

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		expectedMsg := fmt.Sprintf("Entity %v of type %v not found",
			dlName,
			registry.DistributionListType,
		)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, resp["error"], expectedMsg)
	})
}
