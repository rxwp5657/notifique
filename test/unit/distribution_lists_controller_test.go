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

	di "github.com/notifique/dependency_injection"
	"github.com/notifique/dto"
	"github.com/notifique/internal"
	mk "github.com/notifique/test/mocks"
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

	testCreateDistributionList(t, testApp.Engine, *testApp.Storage.MockDistributionListStorage)
	testAddRecipients(t, testApp.Engine, *testApp.Storage.MockDistributionListStorage)
	testDeleteRecipients(t, testApp.Engine, *testApp.Storage.MockDistributionListStorage)
}

func testCreateDistributionList(t *testing.T, e *gin.Engine, mock mk.MockDistributionListStorage) {

	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	createDistributionList := func(dl dto.DistributionList) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()

		marshalled, _ := json.Marshal(dl)
		reader := bytes.NewReader(marshalled)

		req, _ := http.NewRequest("POST", distributionListUrl, reader)
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
			Return(internal.DistributionListAlreadyExists{
				Name: dl.Name,
			})

		w := createDistributionList(dl)

		resp := make(map[string]string, 0)

		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.FailNow()
		}

		errTemplate := "Distribution list %s already exists"
		expectedMsg := fmt.Sprintf(errTemplate, dl.Name)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, expectedMsg, resp["error"])
	})

	t.Run("Should fail if the distribution list name is less than the allowed min", func(t *testing.T) {

	})

	t.Run("Should fail if the distribution list name is greather than the allowed max", func(t *testing.T) {

	})

	t.Run("Should fail if the distribution list name doesn't comply whith the name spec", func(t *testing.T) {

	})

	t.Run("Should fail if the number of recipients is greather than the maximum allowed", func(t *testing.T) {

	})

	t.Run("Should fail if there are duplicated recipients", func(t *testing.T) {

	})
}

func testAddRecipients(t *testing.T, e *gin.Engine, mock mk.MockDistributionListStorage) {

	// addRecipients := func(dlName string, recipients []string) *httptest.ResponseRecorder {

	// 	w := httptest.NewRecorder()

	// 	body := make(map[string][]string)
	// 	body["recipients"] = recipients

	// 	marshalled, _ := json.Marshal(body)
	// 	reader := bytes.NewReader(marshalled)

	// 	url := fmt.Sprintf("%s/%s/recipients", distributionListUrl, dlName)

	// 	req, _ := http.NewRequest("PATCH", url, reader)
	// 	req.Header.Add("userId", testUserId)

	// 	e.ServeHTTP(w, req)

	// 	return w
	// }

}

func testDeleteRecipients(t *testing.T, e *gin.Engine, mock mk.MockDistributionListStorage) {

	// deleteRecipients := func(dlName string, recipients []string) *httptest.ResponseRecorder {

	// 	w := httptest.NewRecorder()

	// 	body := make(map[string][]string)
	// 	body["recipients"] = recipients

	// 	marshalled, _ := json.Marshal(body)
	// 	reader := bytes.NewReader(marshalled)

	// 	url := fmt.Sprintf("%s/%s/recipients", distributionListUrl, dlName)

	// 	req, _ := http.NewRequest("DELETE", url, reader)
	// 	req.Header.Add("userId", testUserId)

	// 	e.ServeHTTP(w, req)

	// 	return w
	// }
}
