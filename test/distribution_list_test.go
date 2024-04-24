package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/gin-gonic/gin"
	c "github.com/notifique/controllers"
	"github.com/notifique/dto"
	"github.com/notifique/routes"
	"github.com/stretchr/testify/assert"
)

func makeDistributionListRouter(dls c.DistributionListStorage) *gin.Engine {
	r := gin.Default()
	routes.SetupDistributionListRoutes(r, dls)
	return r
}

func makeRecipientsList(numRecipients int) []string {
	recipients := make([]string, 0, numRecipients)

	for i := range numRecipients {
		recipients = append(recipients, fmt.Sprint(i))
	}

	return recipients
}

func TestCreateDistributionList(t *testing.T) {
	storage := getStorage()
	router := makeDistributionListRouter(&storage)

	dl := dto.DistributionList{
		Name:       "TestDL",
		Recipients: []string{"1", "2", "123"},
	}

	w := httptest.NewRecorder()

	marshalled, _ := json.Marshal(dl)
	reader := bytes.NewReader(marshalled)

	req, _ := http.NewRequest("POST", "/v0/distribution-lists", reader)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
}

func TestCreateDuplicatedDistributionList(t *testing.T) {
	storage := getStorage()
	router := makeDistributionListRouter(&storage)

	originalDL := dto.DistributionList{
		Name:       "TestDL",
		Recipients: []string{"1", "2", "123"},
	}

	template := "Distribution list %v already exists"
	expectedError := fmt.Sprintf(template, originalDL.Name)

	ctx := context.Background()
	storage.CreateDistributionList(ctx, originalDL)

	dl := dto.DistributionList{
		Name:       originalDL.Name,
		Recipients: []string{},
	}

	w := httptest.NewRecorder()

	marshalled, _ := json.Marshal(dl)
	reader := bytes.NewReader(marshalled)

	req, _ := http.NewRequest("POST", "/v0/distribution-lists", reader)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	resp := make(map[string]string)

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.FailNow()
	}

	assert.Equal(t, 400, w.Code)
	assert.Equal(t, expectedError, resp["error"])
}

func TestGetDistributionLists(t *testing.T) {
	storage := getStorage()
	router := makeDistributionListRouter(&storage)

	lists := []dto.DistributionList{
		{Name: "TestDL1", Recipients: []string{"1", "2", "123"}},
		{Name: "TestDL2", Recipients: []string{"1"}},
		{Name: "TestDL3", Recipients: []string{}},
	}

	expectedResp := []dto.DistributionListSummary{
		{Name: lists[0].Name, NumberOfRecipients: len(lists[0].Recipients)},
		{Name: lists[1].Name, NumberOfRecipients: len(lists[1].Recipients)},
		{Name: lists[2].Name, NumberOfRecipients: len(lists[2].Recipients)},
	}

	ctx := context.Background()

	for _, list := range lists {
		storage.CreateDistributionList(ctx, list)
	}

	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/v0/distribution-lists", nil)
	req.Header.Add("userId", userId)

	router.ServeHTTP(w, req)

	resp := make([]dto.DistributionListSummary, 0)

	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.FailNow()
	}

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, expectedResp, resp)
}

func TestGetDistributionListRecipients(t *testing.T) {

	storage := getStorage()
	router := makeDistributionListRouter(&storage)

	dl := dto.DistributionList{
		Name:       "TestDL",
		Recipients: []string{"1", "2", "123"},
	}

	sortedRecipients := make([]string, len(dl.Recipients))
	copy(sortedRecipients, dl.Recipients)
	sort.Slice(sortedRecipients, func(i, j int) bool {
		return sortedRecipients[i] < sortedRecipients[j]
	})

	ctx := context.Background()
	storage.CreateDistributionList(ctx, dl)

	getRecipients := func(dlName string, filters *dto.PageFilter) *httptest.ResponseRecorder {

		w := httptest.NewRecorder()

		url := fmt.Sprintf("/v0/distribution-lists/%v/recipients", dlName)

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("userId", userId)

		if filters != nil {
			q := req.URL.Query()
			if filters.Skip != nil {
				q.Add("skip", fmt.Sprint(*filters.Skip))
			}
			if filters.Take != nil {
				q.Add("take", fmt.Sprint(*filters.Take))
			}

			req.URL.RawQuery = q.Encode()
		}

		router.ServeHTTP(w, req)

		return w
	}

	t.Run("Should be able to retrieve the recipients", func(t *testing.T) {

		w := getRecipients(dl.Name, nil)

		resp := make([]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, sortedRecipients, resp)
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

		assert.Equal(t, 404, w.Code)
		assert.Equal(t, expectedError, resp["error"])
	})

	t.Run("Should be able to apply filters", func(t *testing.T) {
		take, skip := 1, 2

		filters := dto.PageFilter{
			Skip: &skip,
			Take: &take,
		}

		w := getRecipients(dl.Name, &filters)

		resp := make([]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, []string{"2"}, resp)
	})
}

func TestAddRecipeints(t *testing.T) {

	storage := getStorage()
	router := makeDistributionListRouter(&storage)

	dl := dto.DistributionList{
		Name:       "TestDL",
		Recipients: []string{"1", "2", "123"},
	}

	ctx := context.Background()
	storage.CreateDistributionList(ctx, dl)

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

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, expectedSummary, resp)
	})

	t.Run("Should fail if the distribution list doesn't exists", func(t *testing.T) {

		w := addRecipients("Missing", []string{})

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail when adding empty recipients", func(t *testing.T) {
		newRecipients := []string{""}

		w := addRecipients(dl.Name, newRecipients)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail when exceeding the number of recipients", func(t *testing.T) {
		newRecipients := makeRecipientsList(260)

		w := addRecipients(dl.Name, newRecipients)

		assert.Equal(t, 400, w.Code)
	})
}

func TestRemoveRecipeints(t *testing.T) {

	storage := getStorage()
	router := makeDistributionListRouter(&storage)

	dl := dto.DistributionList{
		Name:       "TestDL",
		Recipients: []string{"1", "2", "123"},
	}

	ctx := context.Background()
	storage.CreateDistributionList(ctx, dl)

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
		w := deleteRecipients("Missing", []string{})

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail when adding empty recipients", func(t *testing.T) {
		toDelete := []string{""}

		w := deleteRecipients(dl.Name, toDelete)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("Should fail when exceeding the number of recipients", func(t *testing.T) {
		toDelete := makeRecipientsList(260)

		w := deleteRecipients(dl.Name, toDelete)

		assert.Equal(t, 400, w.Code)
	})
}
