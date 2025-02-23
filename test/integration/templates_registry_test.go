package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/notifique/internal/server/controllers"
	"github.com/notifique/internal/server/dto"
	"github.com/notifique/internal/testutils"
	r "github.com/notifique/internal/testutils/registry"
)

type TestNotificationTemplateRegistry interface {
	controllers.NotificationTemplateRegistry
	r.ContainerTester
	GetNotificationTemplate(ctx context.Context, id string) (dto.NotificationTemplateReq, error)
}

func TestNotificationsTemplatePostgres(t *testing.T) {

	ctx := context.Background()
	tester, close, err := r.NewPostgresIntegrationTester(ctx)

	if err != nil {
		t.Fatal("failed to init postgres tester - ", err)
	}

	defer close()

	testSaveNotificationTemplate(ctx, t, tester)
}

func TestNotificationsTemplateDynamo(t *testing.T) {

	ctx := context.Background()
	tester, close, err := r.NewDynamoRegistryTester(ctx)

	if err != nil {
		t.Fatal("failed to init dynamo tester - ", err)
	}

	defer close()

	testSaveNotificationTemplate(ctx, t, tester)
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
