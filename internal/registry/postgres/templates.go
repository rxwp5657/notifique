package postgresresgistry

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/notifique/internal/server/dto"
)

const insertNotificationTemplate = `
INSERT INTO notification_templates (
	name,
	title_template,
	contents_template,
	description,
	created_by,
	created_at
) VALUES (
	@name,
	@titleTemplate,
	@contentsTemplate,
	@description,
	@createdBy,
	@createdAt
) RETURNING
	id;
`

const insertNotificationTemplateVariables = `
INSERT INTO notification_template_variables (
	template_id,
	name,
	type,
	required,
	validation
) VALUES (
	@templateId,
	@name,
	@type,
	@required,
	@validation
);
`

func (r *Registry) SaveTemplate(ctx context.Context, createdBy string, ntr dto.NotificationTemplateReq) (dto.NotificationTemplateCreatedResp, error) {

	resp := dto.NotificationTemplateCreatedResp{}

	tx, err := r.conn.Begin(ctx)

	if err != nil {
		return resp, fmt.Errorf("failed to start transaction - %w", err)
	}

	createdAt := time.Now().Format(time.RFC3339)

	args := pgx.NamedArgs{
		"name":             ntr.Name,
		"titleTemplate":    ntr.TitleTemplate,
		"description":      ntr.Description,
		"contentsTemplate": ntr.ContentsTemplate,
		"createdBy":        createdBy,
		"createdAt":        createdAt,
	}

	templateId := ""

	err = tx.QueryRow(ctx, insertNotificationTemplate, args).Scan(&templateId)

	if err != nil {
		tx.Rollback(ctx)
		return resp, fmt.Errorf("failed to insert template - %w", err)
	}

	variableArgs := make([]pgx.NamedArgs, 0, len(ntr.Variables))

	for _, v := range ntr.Variables {
		variableArgs = append(variableArgs, pgx.NamedArgs{
			"templateId": templateId,
			"name":       v.Name,
			"type":       v.Type,
			"required":   v.Required,
			"validation": v.Validation,
		})
	}

	err = batchInsert(
		ctx,
		insertNotificationTemplateVariables,
		variableArgs,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return resp, fmt.Errorf("failed to insert template variables - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return resp, fmt.Errorf("failed to commit template insert - %w", err)
	}

	resp.Id = templateId
	resp.Name = ntr.Name
	resp.CreatedAt = createdAt

	return resp, nil
}
