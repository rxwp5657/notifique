package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/notifique/dto"
	storage "github.com/notifique/internal/storage/dynamodb"
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

func TestDistributionListController(t *testing.T) {
	container, err := setupDynamoDB(context.TODO())

	if err != nil {
		t.Fatalf("failed to create container - %s", err)
	}

	client := storage.MakeClient(&container.URI)
	storage := storage.MakeDynamoDBStorage(client)

	router := gin.Default()
	routes.SetupDistributionListRoutes(router, &storage)

	userId := "1234"

	dl := dto.DistributionList{
		Name:       "TestDL",
		Recipients: []string{"1", "2", "123"},
	}

	createDistributionList := func(dl dto.DistributionList) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()

		marshalled, _ := json.Marshal(dl)
		reader := bytes.NewReader(marshalled)

		req, _ := http.NewRequest("POST", "/v0/distribution-lists", reader)
		req.Header.Add("userId", userId)

		router.ServeHTTP(w, req)

		return w
	}

	addRecipients := func(dlName string, recipients []string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		body := make(map[string][]string)
		body["recipients"] = recipients

		marshalled, _ := json.Marshal(body)
		reader := bytes.NewReader(marshalled)

		url := fmt.Sprintf("/v0/distribution-lists/%v/recipients", dlName)

		req, _ := http.NewRequest("PATCH", url, reader)
		req.Header.Add("userId", userId)

		router.ServeHTTP(w, req)

		return w
	}

	deleteRecipients := func(dlName string, recipients []string) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		body := make(map[string][]string)
		body["recipients"] = recipients

		marshalled, _ := json.Marshal(body)
		reader := bytes.NewReader(marshalled)

		url := fmt.Sprintf("/v0/distribution-lists/%v/recipients", dlName)

		req, _ := http.NewRequest("DELETE", url, reader)
		req.Header.Add("userId", userId)

		router.ServeHTTP(w, req)

		return w
	}

	t.Run("TestCreateDistributionList", func(t *testing.T) {

		t.Run("Should be able to create a distribution list", func(t *testing.T) {
			w := createDistributionList(dl)
			assert.Equal(t, http.StatusCreated, w.Code)
		})

		t.Run("Should fail if distribution list has a bad name", func(t *testing.T) {
			dl := dto.DistributionList{
				Name:       "TestDL/BadToken",
				Recipients: []string{"1", "2", "123"},
			}

			w := createDistributionList(dl)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			expectedMsg := "Error:Field validation for 'Name' failed"

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, resp["error"], expectedMsg)
		})

		storage.DeleteDistributionList(context.TODO(), dl.Name)
	})

	t.Run("TestDuplicatedDistributionList", func(t *testing.T) {

		err := storage.CreateDistributionList(context.TODO(), dl)

		if err != nil {
			t.Fatal("failed to create distribution list")
		}

		t.Run("Should not be able to create a list with the same name", func(t *testing.T) {
			w := createDistributionList(dl)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errTemplate := "Distribution list %v already exists"
			expectedMsg := fmt.Sprintf(errTemplate, dl.Name)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, expectedMsg, resp["error"])
		})

		err = storage.DeleteDistributionList(context.TODO(), dl.Name)

		if err != nil {
			t.Fatal("failed to delete distribution list")
		}
	})

	t.Run("TestGetDistributionLists", func(t *testing.T) {

		getDistributionLists := func(userId string, filters *dto.PageFilter) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()

			req, _ := http.NewRequest("GET", "/v0/distribution-lists", nil)
			req.Header.Add("userId", userId)

			addPaginationFilters(req, filters)

			router.ServeHTTP(w, req)

			return w
		}

		distributionLists, err := crateTestDistributionLists(3, &storage)

		if err != nil {
			t.Fatalf("failed to create distribution lists - %v", err)
		}

		t.Run("Should be able to get the default page", func(t *testing.T) {

			expectedResp := dto.Page[dto.DistributionListSummary]{
				NextToken:   nil,
				PrevToken:   nil,
				ResultCount: len(distributionLists),
				Data:        makeSummaries(distributionLists),
			}

			w := getDistributionLists(userId, nil)

			resp := dto.Page[dto.DistributionListSummary]{}

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			assert.Equal(t, http.StatusOK, w.Code)
			assert.ElementsMatch(t, expectedResp.Data, resp.Data)
		})

		t.Run("Can apply pagination filters", func(t *testing.T) {

			maxResults := 1

			pageFilters := dto.PageFilter{
				NextToken:  nil,
				MaxResults: &maxResults,
			}

			pages := make([]dto.Page[dto.DistributionListSummary], 0)

			for {
				w := getDistributionLists(userId, &pageFilters)

				resp := dto.Page[dto.DistributionListSummary]{}

				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal("failed to unmarshall distribution list page")
				}

				if resp.NextToken == nil {
					break
				}

				pages = append(pages, resp)
				pageFilters.NextToken = resp.NextToken
			}

			assert.Equal(t, len(distributionLists), len(pages))
		})

		err = deleteDistributionLists(distributionLists, &storage)

		if err != nil {
			t.Fatalf("failed to delete distribution lists - %v", err)
		}
	})

	t.Run("TestGetRecipients", func(t *testing.T) {

		getRecipients := func(dlName string, filters *dto.PageFilter) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()

			url := fmt.Sprintf("/v0/distribution-lists/%v/recipients", dlName)

			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("userId", userId)

			addPaginationFilters(req, filters)

			router.ServeHTTP(w, req)

			return w
		}

		err := storage.CreateDistributionList(context.TODO(), dl)

		if err != nil {
			t.Fatal("failed to create distribution list")
		}

		t.Run("Should be able to retrieve the recipients", func(t *testing.T) {

			expectedResponse := dto.Page[string]{
				NextToken:   nil,
				PrevToken:   nil,
				ResultCount: len(dl.Recipients),
				Data:        dl.Recipients,
			}

			w := getRecipients(dl.Name, nil)

			resp := dto.Page[string]{}

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			assert.Equal(t, 200, w.Code)
			assert.Nil(t, resp.NextToken)
			assert.Nil(t, resp.PrevToken)
			assert.Equal(t, expectedResponse.ResultCount, resp.ResultCount)
			assert.ElementsMatch(t, expectedResponse.Data, resp.Data)
		})

		t.Run("Should fail if the distribution list doesn't exists", func(t *testing.T) {

			missingDL := "Missing"
			w := getRecipients(missingDL, nil)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errorTemplate := "Distribution list %v not found"
			expectedError := fmt.Sprintf(errorTemplate, missingDL)

			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.Equal(t, expectedError, resp["error"])
		})

		t.Run("Should be able to apply filters", func(t *testing.T) {

			maxResults := 1
			filters := dto.PageFilter{
				NextToken:  nil,
				MaxResults: &maxResults,
			}

			pages := make([]dto.Page[string], 0)

			for {
				w := getRecipients(dl.Name, &filters)

				resp := dto.Page[string]{}

				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatal("failed to unmarshall recipients")
				}

				if resp.NextToken == nil {
					break
				}

				pages = append(pages, resp)
				filters.NextToken = resp.NextToken
			}

			assert.Equal(t, len(dl.Recipients), len(pages))
		})

		err = storage.DeleteDistributionList(context.TODO(), dl.Name)

		if err != nil {
			t.Fatal("failed to delete distribution list")
		}
	})

	t.Run("TestAddRecipients", func(t *testing.T) {

		err := storage.CreateDistributionList(context.TODO(), dl)

		if err != nil {
			t.Fatalf("failed to create distribution list - %v", err)
		}

		t.Run("Should be able to add recipients", func(t *testing.T) {

			newRecipients := []string{"3", "4", "123"}

			expectedSummary := dto.DistributionListSummary{
				Name:               dl.Name,
				NumberOfRecipients: 5,
			}

			w := addRecipients(dl.Name, newRecipients)

			resp := dto.DistributionListSummary{}

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, expectedSummary, resp)
		})

		t.Run("Should fail if the distribution list doesn't exists", func(t *testing.T) {

			dlName := "Missing"

			w := addRecipients("Missing", []string{})

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errTemplate := "Distribution list %v not found"
			expectedMsg := fmt.Sprintf(errTemplate, dlName)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, expectedMsg, resp["error"])
		})

		t.Run("Should fail when adding empty recipients", func(t *testing.T) {
			newRecipients := []string{""}

			w := addRecipients(dl.Name, newRecipients)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errMsg := "Error:Field validation for 'Recipients[0]'"
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, resp["error"], errMsg)
		})

		t.Run("Should fail when exceeding the number of recipients", func(t *testing.T) {
			newRecipients := makeRecipientsList(260)

			w := addRecipients(dl.Name, newRecipients)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errMsg := "validation for 'Recipients' failed on the 'max' tag"
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, resp["error"], errMsg)
		})

		err = storage.DeleteDistributionList(context.TODO(), dl.Name)

		if err != nil {
			t.Fatalf("failed to delete distribution list - %v", err)
		}
	})

	t.Run("TestRemoveRecipients", func(t *testing.T) {

		err := storage.CreateDistributionList(context.TODO(), dl)

		if err != nil {
			t.Fatalf("failed to create distribution list - %v", err)
		}

		t.Run("Should be able to delete recipients", func(t *testing.T) {
			toDelete := []string{"1", "2"}

			expectedSummary := dto.DistributionListSummary{
				Name:               dl.Name,
				NumberOfRecipients: 1,
			}

			w := deleteRecipients(dl.Name, toDelete)

			resp := dto.DistributionListSummary{}

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			assert.Equal(t, 200, w.Code)
			assert.Equal(t, expectedSummary, resp)
		})

		t.Run("Should fail if the distribution list doesn't exists", func(t *testing.T) {
			dlName := "Missing"

			w := deleteRecipients(dlName, []string{})

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errTemplate := "Distribution list %v not found"
			errMsg := fmt.Sprintf(errTemplate, dlName)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, errMsg, resp["error"])
		})

		t.Run("Should fail when adding empty recipients", func(t *testing.T) {
			toDelete := []string{""}

			w := deleteRecipients(dl.Name, toDelete)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errMsg := "Error:Field validation for 'Recipients[0]'"
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, resp["error"], errMsg)
		})

		t.Run("Should fail when exceeding the number of recipients", func(t *testing.T) {
			toDelete := makeRecipientsList(260)

			w := deleteRecipients(dl.Name, toDelete)

			resp := make(map[string]string, 0)

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.FailNow()
			}

			errMsg := "validation for 'Recipients' failed on the 'max' tag"
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, resp["error"], errMsg)
		})

		err = storage.DeleteDistributionList(context.TODO(), dl.Name)

		if err != nil {
			t.Fatalf("failed to delete distribution list - %v", err)
		}
	})

	t.Run("TestDuplicatedRecipients", func(t *testing.T) {

		err := storage.CreateDistributionList(context.TODO(), dl)

		if err != nil {
			t.Fatalf("failed to create distribution list - %v", err)
		}

		t.Run("Should do nothing if users are already added", func(t *testing.T) {
			recipients := dl.Recipients[:2]

			expectedSummary := dto.DistributionListSummary{
				Name:               dl.Name,
				NumberOfRecipients: len(dl.Recipients),
			}

			w := addRecipients(dl.Name, recipients)

			resp := dto.DistributionListSummary{}

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatal("failed to unmarshall dist list summary body")
			}

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, expectedSummary, resp)
		})

		t.Run("Should do nothing if users are not recipients of dl", func(t *testing.T) {
			recipients := []string{"-1", "-2"}

			expectedSummary := dto.DistributionListSummary{
				Name:               dl.Name,
				NumberOfRecipients: len(dl.Recipients),
			}

			w := deleteRecipients(dl.Name, recipients)

			resp := dto.DistributionListSummary{}

			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatal("failed to unmarshall dist list summary body")
			}

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, expectedSummary, resp)
		})

		err = storage.DeleteDistributionList(context.TODO(), dl.Name)

		if err != nil {
			t.Fatalf("failed to delete distribution list - %v", err)
		}
	})

	t.Run("TestDeleteDistributionList", func(t *testing.T) {

		dl := dto.DistributionList{
			Name:       "TestDL",
			Recipients: []string{"1", "2", "123"},
		}

		err := storage.CreateDistributionList(context.TODO(), dl)

		if err != nil {
			t.Fatalf("failed to create distribution list - %v", err)
		}

		w := httptest.NewRecorder()

		url := fmt.Sprintf("/v0/distribution-lists/%v", dl.Name)

		req, _ := http.NewRequest("DELETE", url, nil)
		req.Header.Add("userId", userId)

		router.ServeHTTP(w, req)

		assert.Equal(t, 204, w.Code)
	})
}
