package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/notifique/internal/registry"
	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/dto"
	"github.com/notifique/internal/testutils"
	r "github.com/notifique/internal/testutils/registry"
)

type DistributionListTester interface {
	controllers.DistributionRegistry
	r.ContainerTester
	GetDistributionList(ctx context.Context, dlName string) (dto.DistributionList, error)
	DistributionListExists(ctx context.Context, dlName string) (bool, error)
}

func TestDistributionRegistryPostgres(t *testing.T) {
	ctx := context.Background()
	tester, close, err := r.NewPostgresIntegrationTester(ctx)

	if err != nil {
		t.Fatal("failed to init postgres tester - ", err)
	}

	defer close()

	testCreateDistributionList(ctx, t, tester)
	testGetDistributionListsSummaries(ctx, t, tester)
	testDeleteDistributionList(ctx, t, tester)
	testGetDistributionListRecipients(ctx, t, tester)
	testAddRecipients(ctx, t, tester)
	testAddRecipientsThatAreOnTheDL(ctx, t, tester)
	testRemoveRecipients(ctx, t, tester)
	testDeleteRecipientsThatAreNotOnDL(ctx, t, tester)
}

func TestDistributionListRegistryDynamo(t *testing.T) {
	ctx := context.Background()
	tester, close, err := r.NewDynamoRegistryTester(ctx)

	if err != nil {
		t.Fatal("failed to init dynamo tester - ", err)
	}

	defer close()

	testCreateDistributionList(ctx, t, tester)
	testGetDistributionListsSummaries(ctx, t, tester)
	testDeleteDistributionList(ctx, t, tester)
	testGetDistributionListRecipients(ctx, t, tester)
	testAddRecipients(ctx, t, tester)
	testAddRecipientsThatAreOnTheDL(ctx, t, tester)
	testRemoveRecipients(ctx, t, tester)
	testDeleteRecipientsThatAreNotOnDL(ctx, t, tester)
}

func setupTestDL(ctx context.Context, t *testing.T, dlt DistributionListTester) dto.DistributionList {
	t.Helper()

	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	err := dlt.CreateDistributionList(ctx, dl)

	if err != nil {
		t.Fatal(fmt.Errorf("failed to insert test distribution list - %w", err))
	}

	return dl
}

func testCreateDistributionList(ctx context.Context, t *testing.T, dlt DistributionListTester) {

	dl := dto.DistributionList{
		Name:       "Test",
		Recipients: []string{"1", "2", "3"},
	}

	t.Run("Can create distribution list", func(t *testing.T) {
		err := dlt.CreateDistributionList(context.TODO(), dl)
		assert.Nil(t, err)

		newDL, err := dlt.GetDistributionList(ctx, dl.Name)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to retrieve inserted distribution list - %w", err))
		}

		assert.Equal(t, dl, newDL)
	})

	r.Clear(ctx, t, dlt)

	t.Run("Should fail if the distribution list already exists", func(t *testing.T) {
		err := dlt.CreateDistributionList(context.TODO(), dl)

		if err != nil {
			t.Fatal("failed to insert the distribution list - %w", err)
		}

		err = dlt.CreateDistributionList(context.TODO(), dl)

		assert.ErrorAs(t, err, &server.DistributionListAlreadyExists{Name: dl.Name})
	})

	r.Clear(ctx, t, dlt)
}

func testGetDistributionListsSummaries(ctx context.Context, t *testing.T, dlt DistributionListTester) {

	testDLs := testutils.MakeDistributionLists(3)
	testSummaries := testutils.MakeSummaries(testDLs)

	for _, dl := range testDLs {
		err := dlt.CreateDistributionList(ctx, dl)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to insert test distribution list - %w", err))
		}
	}

	defer r.Clear(ctx, t, dlt)

	t.Run("Can retrieve a page of distribution lists summaries", func(t *testing.T) {
		pageFilters := dto.PageFilter{}
		summaries, err := dlt.GetDistributionLists(ctx, pageFilters)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to retrieve distribution lists summaries - %w", err))
		}

		assert.Nil(t, summaries.NextToken)
		assert.Nil(t, summaries.PrevToken)
		assert.Equal(t, len(testSummaries), summaries.ResultCount)
		assert.ElementsMatch(t, testSummaries, summaries.Data)
	})

	t.Run("Can paginate distribution list summaries", func(t *testing.T) {

		maxResults := 1

		pageFilters := dto.PageFilter{
			MaxResults: &maxResults,
		}

		summaries := make([]dto.DistributionListSummary, 0, len(testDLs))

		for {
			summariesPage, err := dlt.GetDistributionLists(ctx, pageFilters)

			if err != nil {
				t.Fatal(fmt.Errorf("failed to retrieve distribution lists summaries - %w", err))
			}

			if len(summariesPage.Data) == 0 {
				break
			}

			summaries = append(summaries, summariesPage.Data...)

			pageFilters.NextToken = summariesPage.NextToken
		}

		assert.ElementsMatch(t, testSummaries, summaries)
	})
}

func testDeleteDistributionList(ctx context.Context, t *testing.T, dlt DistributionListTester) {

	dl := setupTestDL(ctx, t, dlt)

	defer r.Clear(ctx, t, dlt)

	t.Run("Can delete a distribution list with all of its recipients", func(t *testing.T) {
		err := dlt.DeleteDistributionList(ctx, dl.Name)
		assert.Nil(t, err)

		exists, err := dlt.DistributionListExists(ctx, dl.Name)

		if err != nil {
			t.Fatal(err)
		}

		assert.False(t, exists)
	})
}

func testGetDistributionListRecipients(ctx context.Context, t *testing.T, dlt DistributionListTester) {

	dl := setupTestDL(ctx, t, dlt)

	defer r.Clear(ctx, t, dlt)

	t.Run("Can retrieve the recipients of a distribution list", func(t *testing.T) {
		pageFilters := dto.PageFilter{}
		recipientsPage, err := dlt.GetRecipients(ctx, dl.Name, pageFilters)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to retrieve recipients page - %w", err))
		}

		assert.Nil(t, recipientsPage.PrevToken)
		assert.Nil(t, recipientsPage.NextToken)
		assert.Equal(t, len(dl.Recipients), recipientsPage.ResultCount)
		assert.ElementsMatch(t, dl.Recipients, recipientsPage.Data)
	})

	t.Run("Can paginate the recipients of a distribution list", func(t *testing.T) {

		maxResults := 1
		pageFilters := dto.PageFilter{MaxResults: &maxResults}

		recipients := make([]string, 0, len(dl.Recipients))

		for {
			recipientsPage, err := dlt.GetRecipients(ctx, dl.Name, pageFilters)

			if err != nil {
				t.Fatal(fmt.Errorf("failed to retrieve recipients page - %w", err))
			}

			if recipientsPage.ResultCount == 0 {
				break
			}

			recipients = append(recipients, recipientsPage.Data...)
			pageFilters.NextToken = recipientsPage.NextToken
		}

		assert.ElementsMatch(t, dl.Recipients, recipients)
	})

	t.Run("Should fail if we try to get the recipients of a DL that doesn't exist", func(t *testing.T) {
		dlName := "Missing Distribution List"
		pageFilters := dto.PageFilter{}
		_, err := dlt.GetRecipients(ctx, dlName, pageFilters)
		assert.ErrorAs(t, err, &server.EntityNotFound{Id: dlName, Type: registry.DistributionListType})
	})
}

func testAddRecipients(ctx context.Context, t *testing.T, dlt DistributionListTester) {

	dl := setupTestDL(ctx, t, dlt)

	defer r.Clear(ctx, t, dlt)

	newRecipients := []string{"4", "5", "6"}

	t.Run("Can add new recipients to the distribution list", func(t *testing.T) {
		summary, err := dlt.AddRecipients(ctx, dl.Name, newRecipients)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to add new recipients to the distribution list - %w", err))
		}

		assert.Equal(t, len(dl.Recipients)+len(newRecipients), summary.NumberOfRecipients)

		pageFilters := dto.PageFilter{}
		recipientsPage, err := dlt.GetRecipients(ctx, dl.Name, pageFilters)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to retrieve recipients page - %w", err))
		}

		fullRecipients := append(dl.Recipients, newRecipients...)
		assert.ElementsMatch(t, fullRecipients, recipientsPage.Data)
	})

	t.Run("Should fail when trying to add recipients to a DL that doesn't exist", func(t *testing.T) {
		dlName := "Missing Distribution List"
		_, err := dlt.AddRecipients(ctx, dlName, newRecipients)
		assert.ErrorAs(t, err, &server.EntityNotFound{Id: dlName, Type: registry.DistributionListType})
	})
}

func testAddRecipientsThatAreOnTheDL(ctx context.Context, t *testing.T, dlt DistributionListTester) {

	dl := setupTestDL(ctx, t, dlt)

	defer r.Clear(ctx, t, dlt)

	t.Run("Should do nothing when adding users that are on the dl already", func(t *testing.T) {
		summary, err := dlt.AddRecipients(ctx, dl.Name, dl.Recipients)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to add new recipients to the distribution list - %w", err))
		}

		assert.Equal(t, len(dl.Recipients), summary.NumberOfRecipients)

		pageFilters := dto.PageFilter{}
		recipientsPage, err := dlt.GetRecipients(ctx, dl.Name, pageFilters)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to retrieve recipients page - %w", err))
		}

		assert.ElementsMatch(t, dl.Recipients, recipientsPage.Data)
	})
}

func testRemoveRecipients(ctx context.Context, t *testing.T, dlt DistributionListTester) {

	dl := setupTestDL(ctx, t, dlt)

	defer r.Clear(ctx, t, dlt)

	recipientsToDelete := []string{"1", "2"}

	t.Run("Can delete recipients from the distribution list", func(t *testing.T) {
		summary, err := dlt.DeleteRecipients(ctx, dl.Name, recipientsToDelete)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to add new recipients to the distribution list - %w", err))
		}

		assert.Equal(t, len(dl.Recipients)-len(recipientsToDelete), summary.NumberOfRecipients)

		pageFilters := dto.PageFilter{}
		recipientsPage, err := dlt.GetRecipients(ctx, dl.Name, pageFilters)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to retrieve recipients page - %w", err))
		}

		assert.ElementsMatch(t, []string{"3"}, recipientsPage.Data)
	})

	t.Run("Should fail when trying to delete recipients of a DL that doesn't exist", func(t *testing.T) {
		dlName := "Missing Distribution List"
		_, err := dlt.DeleteRecipients(ctx, dlName, recipientsToDelete)
		assert.ErrorAs(t, err, &server.EntityNotFound{Id: dlName, Type: registry.DistributionListType})
	})
}

func testDeleteRecipientsThatAreNotOnDL(ctx context.Context, t *testing.T, dlt DistributionListTester) {

	dl := setupTestDL(ctx, t, dlt)

	defer r.Clear(ctx, t, dlt)

	t.Run("Should do nothing if when deleting recipients that are not on the dl", func(t *testing.T) {

		summary, err := dlt.DeleteRecipients(ctx, dl.Name, []string{"4", "5", "6"})

		if err != nil {
			t.Fatal(fmt.Errorf("failed to add new recipients to the distribution list - %w", err))
		}

		assert.Equal(t, len(dl.Recipients), summary.NumberOfRecipients)

		pageFilters := dto.PageFilter{}
		recipientsPage, err := dlt.GetRecipients(ctx, dl.Name, pageFilters)

		if err != nil {
			t.Fatal(fmt.Errorf("failed to retrieve recipients page - %w", err))
		}

		assert.ElementsMatch(t, dl.Recipients, recipientsPage.Data)
	})
}
