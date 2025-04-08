package integration_test

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/notifique/service/internal"
	"github.com/notifique/service/internal/controllers"
	"github.com/notifique/service/internal/dto"
	"github.com/notifique/service/internal/registry"
	"github.com/notifique/service/internal/testutils"
	r "github.com/notifique/service/internal/testutils/registry"
	sdto "github.com/notifique/shared/dto"
)

type TestNotificationTemplateRegistry interface {
	controllers.NotificationTemplateRegistry
	r.ContainerTester
	GetNotificationTemplate(ctx context.Context, id string) (dto.NotificationTemplateReq, error)
	TemplateExists(ctx context.Context, id string) (bool, error)
}

func TestNotificationsTemplatePostgres(t *testing.T) {

	ctx := context.Background()
	tester, close, err := r.NewPostgresIntegrationTester(ctx)

	if err != nil {
		t.Fatal("failed to init postgres tester - ", err)
	}

	defer close()

	testSaveNotificationTemplate(ctx, t, tester)
	testGetNotificationTemplates(ctx, t, tester)
	testGetNotificationTemplateDetails(ctx, t, tester)
	testDeleteNotificationTemplate(ctx, t, tester)
}

func TestNotificationsTemplateDynamo(t *testing.T) {

	ctx := context.Background()
	tester, close, err := r.NewDynamoRegistryTester(ctx)

	if err != nil {
		t.Fatal("failed to init dynamo tester - ", err)
	}

	defer close()

	testSaveNotificationTemplate(ctx, t, tester)
	testGetNotificationTemplates(ctx, t, tester)
	testGetNotificationTemplateDetails(ctx, t, tester)
	testDeleteNotificationTemplate(ctx, t, tester)
}

func testSaveNotificationTemplate(ctx context.Context, t *testing.T, ntr TestNotificationTemplateRegistry) {

	testUser := "1234"

	defer ntr.ClearDB(ctx)

	t.Run("Can save a notification template", func(t *testing.T) {
		req := testutils.MakeTestNotificationTemplateRequest()

		resp, err := ntr.SaveTemplate(ctx, testUser, req)

		_, idErr := uuid.Parse(resp.Id)

		assert.Nil(t, err)
		assert.Nil(t, idErr)
		assert.NotEmpty(t, resp.Id)
		assert.NotEmpty(t, resp.CreatedAt)
		assert.Equal(t, req.Name, resp.Name)

		template, err := ntr.GetNotificationTemplate(ctx, resp.Id)

		assert.Nil(t, err)
		assert.Equal(t, req, template)

		fmt.Println(template)
	})
}

func testGetNotificationTemplates(ctx context.Context, t *testing.T, ntr TestNotificationTemplateRegistry) {

	defer ntr.ClearDB(ctx)

	testUser := "1234"
	numTemplates := 3
	testTemplates := testutils.MakeTestNotificationTemplateRequests(numTemplates)
	templateInfos := make([]dto.NotificationTemplateInfoResp, 0, len(testTemplates))

	for _, req := range testTemplates {
		template, err := ntr.SaveTemplate(ctx, testUser, req)

		if err != nil {
			t.Fatalf("fail to insert notification template")
		}

		templateInfos = append(templateInfos, dto.NotificationTemplateInfoResp{
			Id:          template.Id,
			Name:        template.Name,
			Description: req.Description,
		})
	}

	sort.Slice(templateInfos, func(i, j int) bool {
		return templateInfos[i].Name < templateInfos[j].Name
	})

	t.Run("Can retrieve a page of notification templates", func(t *testing.T) {
		filters := dto.NotificationTemplateFilters{}

		page, err := ntr.GetTemplates(ctx, filters)

		if err != nil {
			t.Fatalf("failed to retrieve page - %v", err)
		}

		assert.Nil(t, page.NextToken)
		assert.Nil(t, page.PrevToken)
		assert.Equal(t, page.ResultCount, len(templateInfos))
		assert.ElementsMatch(t, templateInfos, page.Data)
	})

	t.Run("Can get all pages of notification templates", func(t *testing.T) {
		maxResults := 1

		filters := dto.NotificationTemplateFilters{
			PageFilter: sdto.PageFilter{MaxResults: &maxResults},
		}

		templates := make([]dto.NotificationTemplateInfoResp, 0, len(templateInfos))

		for {
			page, err := ntr.GetTemplates(ctx, filters)

			if err != nil {
				t.Fatal(fmt.Errorf("failed to retrieve notification templates page - %w", err))
			}

			if page.ResultCount == 0 {
				break
			}

			templates = append(templates, page.Data...)
			filters.NextToken = page.NextToken
		}

		assert.ElementsMatch(t, templateInfos, templates)
	})

	t.Run("Can filter by template name", func(t *testing.T) {
		templateName := fmt.Sprintf(
			testutils.GenericNotificationTemplateName,
			strconv.Itoa(0))

		filters := dto.NotificationTemplateFilters{
			TemplateName: &templateName,
		}

		page, err := ntr.GetTemplates(ctx, filters)

		if err != nil {
			t.Fatalf("failed to retrieve page - %v", err)
		}

		expectedInfos := make([]dto.NotificationTemplateInfoResp, 1)
		copy(expectedInfos, templateInfos[:1])

		assert.Nil(t, page.NextToken)
		assert.Nil(t, page.PrevToken)
		assert.Equal(t, 1, page.ResultCount)
		assert.Equal(t, expectedInfos, page.Data)
	})
}

func testGetNotificationTemplateDetails(ctx context.Context, t *testing.T, ntr TestNotificationTemplateRegistry) {

	defer ntr.ClearDB(ctx)

	testUser := "1234"
	req := testutils.MakeTestNotificationTemplateRequest()

	saved, err := ntr.SaveTemplate(ctx, testUser, req)

	if err != nil {
		t.Fatalf("failed to save template for test - %v", err)
	}

	t.Run("Can retrieve template details", func(t *testing.T) {
		details, err := ntr.GetTemplateDetails(ctx, saved.Id)

		assert.Nil(t, err)
		assert.Equal(t, saved.Id, details.Id)
		assert.Equal(t, req.Name, details.Name)
		assert.Equal(t, req.IsHtml, details.IsHtml)
		assert.Equal(t, req.Description, details.Description)
		assert.Equal(t, req.TitleTemplate, details.TitleTemplate)
		assert.Equal(t, req.ContentsTemplate, details.ContentsTemplate)
		assert.Equal(t, testUser, details.CreatedBy)
		assert.NotEmpty(t, details.CreatedAt)
		assert.Nil(t, details.UpdatedAt)
		assert.Nil(t, details.UpdatedBy)
		assert.ElementsMatch(t, req.Variables, details.Variables)
	})

	t.Run("Returns error for non-existent template", func(t *testing.T) {
		nonExistentId := uuid.New().String()
		_, err := ntr.GetTemplateDetails(ctx, nonExistentId)

		assert.ErrorAs(t, err, &internal.EntityNotFound{
			Id:   nonExistentId,
			Type: registry.NotificationTemplateType,
		})
	})

	t.Run("Returns error for invalid template id", func(t *testing.T) {
		_, err := ntr.GetTemplateDetails(ctx, "invalid-id")

		assert.Error(t, err)
	})
}

func testDeleteNotificationTemplate(ctx context.Context, t *testing.T, ntr TestNotificationTemplateRegistry) {

	defer ntr.ClearDB(ctx)

	testUser := "1234"
	req := testutils.MakeTestNotificationTemplateRequest()

	setupTemplate := func() dto.NotificationTemplateCreatedResp {
		saved, err := ntr.SaveTemplate(ctx, testUser, req)

		if err != nil {
			t.Fatalf("failed to save template for test - %v", err)
		}

		return saved
	}

	t.Run("Can delete a notification template", func(t *testing.T) {
		template := setupTemplate()

		err := ntr.DeleteTemplate(ctx, template.Id)

		assert.Nil(t, err)

		exists, err := ntr.TemplateExists(ctx, template.Id)

		if err != nil {
			t.Fatal(err.Error())
		}

		assert.False(t, exists)
	})

	t.Run("Can delete a notification that is already deleted", func(t *testing.T) {
		template := setupTemplate()

		err := ntr.DeleteTemplate(ctx, template.Id)

		if err != nil {
			t.Fatal(err.Error())
		}

		err = ntr.DeleteTemplate(ctx, template.Id)
		assert.Nil(t, err)
	})

	t.Run("Should do nothing if the template doesn't exist", func(t *testing.T) {
		err := ntr.DeleteTemplate(ctx, uuid.NewString())
		assert.Nil(t, err)
	})
}
