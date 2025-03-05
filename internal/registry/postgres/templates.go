package postgresresgistry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/notifique/internal/registry"
	"github.com/notifique/internal/server"
	"github.com/notifique/internal/server/dto"
)

const insertNotificationTemplate = `
INSERT INTO notification_templates (
	id,
	name,
	title_template,
	contents_template,
	description,
	created_by,
	created_at
) VALUES (
	@id,
	@name,
	@titleTemplate,
	@contentsTemplate,
	@description,
	@createdBy,
	@createdAt
);
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

const getNotificationTemplateInfo = `
SELECT 
	id,
	"name",
	description
FROM
	notification_templates
%s
ORDER BY
	id ASC
LIMIT
	@limit;
`

const getNotificationTemplateDetails = `
SELECT
	id,
	"name",
	title_template,
	contents_template,
	"description",
	created_by,
	created_at,
	updated_by,
	updated_at
FROM
	notification_templates
WHERE
	id = $1;
`

const getTemplateVariables = `
SELECT
	"name",
	"type",
	"required",
	"validation"
FROM
	notification_template_variables
WHERE
	template_id = $1;
`

const deleteTemplateInfo = `
DELETE FROM
	notification_templates
WHERE
	id = $1;
`

type notificationTemplateInfo struct {
	Id          string `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
}

type notificationTemplateKey struct {
	Id   string
	Name *string
}

func (r *Registry) SaveTemplate(ctx context.Context, createdBy string, ntr dto.NotificationTemplateReq) (dto.NotificationTemplateCreatedResp, error) {

	resp := dto.NotificationTemplateCreatedResp{}

	tx, err := r.conn.Begin(ctx)

	if err != nil {
		return resp, fmt.Errorf("failed to start transaction - %w", err)
	}

	id, err := uuid.NewV7()

	if err != nil {
		return resp, fmt.Errorf("failed to create id - %w", err)
	}

	templateId := id.String()
	createdAt := time.Now().Format(time.RFC3339)

	args := pgx.NamedArgs{
		"id":               templateId,
		"name":             ntr.Name,
		"titleTemplate":    ntr.TitleTemplate,
		"description":      ntr.Description,
		"contentsTemplate": ntr.ContentsTemplate,
		"createdBy":        createdBy,
		"createdAt":        createdAt,
	}

	_, err = tx.Exec(ctx, insertNotificationTemplate, args)

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

func (r *Registry) GetTemplates(ctx context.Context, filters dto.NotificationTemplateFilters) (dto.Page[dto.NotificationTemplateInfoResp], error) {

	page := dto.Page[dto.NotificationTemplateInfoResp]{}

	args := pgx.NamedArgs{"limit": server.PageSize}

	whereFilters := make([]string, 0)

	if filters.MaxResults != nil {
		args["limit"] = *filters.MaxResults
	}

	if filters.NextToken != nil {
		nextTokenFilter := "(id) > (@id)"
		whereFilters = append(whereFilters, nextTokenFilter)

		var unmarsalledKey notificationTemplateKey
		err := registry.UnmarshalKey(*filters.NextToken, &unmarsalledKey)

		if err != nil {
			return page, err
		}

		if unmarsalledKey.Name != filters.TemplateName {
			return page, fmt.Errorf("invalid next token %s", *filters.NextToken)
		}

		args["id"] = unmarsalledKey.Id
	}

	if filters.TemplateName != nil {
		whereFilters = append(whereFilters, `"name" LIKE @name`)
		args["name"] = fmt.Sprintf("%s%%", *filters.TemplateName)
	}

	whereStmt := strings.Join(whereFilters, "AND")

	if len(whereStmt) != 0 {
		whereStmt = fmt.Sprintf("WHERE %s", whereStmt)
	}

	query := fmt.Sprintf(getNotificationTemplateInfo, whereStmt)

	rows, err := r.conn.Query(ctx, query, args)

	if err != nil {
		return page, fmt.Errorf("failed to query rows - %w", err)
	}

	defer rows.Close()

	templates, err := pgx.CollectRows(rows, pgx.RowToStructByName[notificationTemplateInfo])

	if err != nil {
		return page, fmt.Errorf("failed to collect rows - %w", err)
	}

	for _, t := range templates {
		page.Data = append(page.Data, dto.NotificationTemplateInfoResp{
			Id:          t.Id,
			Name:        t.Name,
			Description: t.Description,
		})
	}

	numTemplates := len(templates)

	if numTemplates == args["limit"] {
		lastTemplate := templates[numTemplates-1]
		lastTemplateKey := notificationTemplateKey{
			Id:   lastTemplate.Id,
			Name: filters.TemplateName,
		}

		key, err := registry.MarshalKey(lastTemplateKey)

		if err != nil {
			return page, err
		}

		page.NextToken = &key
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = numTemplates

	return page, nil
}

func (r *Registry) GetTemplateDetails(ctx context.Context, templateId string) (dto.NotificationTemplateDetails, error) {

	details := dto.NotificationTemplateDetails{}

	var createdAt time.Time
	var updatedAt *time.Time

	err := r.conn.QueryRow(ctx, getNotificationTemplateDetails, templateId).
		Scan(
			&details.Id,
			&details.Name,
			&details.TitleTemplate,
			&details.ContentsTemplate,
			&details.Description,
			&details.CreatedBy,
			&createdAt,
			&details.UpdatedBy,
			&updatedAt,
		)

	if errors.Is(err, pgx.ErrNoRows) {
		return details, server.EntityNotFound{
			Id:   templateId,
			Type: registry.NotificationTemplateType,
		}
	}

	if err != nil {
		return details, fmt.Errorf("failed to retrieve template details - %w", err)
	}

	details.CreatedAt = createdAt.Format(time.RFC3339)

	if updatedAt != nil {
		updatedAtStr := updatedAt.Format(time.RFC3339)
		details.UpdatedAt = &updatedAtStr
	}

	rows, err := r.conn.Query(ctx, getTemplateVariables, templateId)

	if err != nil {
		return details, fmt.Errorf("failed to query template variables - %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var variable dto.TemplateVariable
		err := rows.Scan(
			&variable.Name,
			&variable.Type,
			&variable.Required,
			&variable.Validation,
		)

		if err != nil {
			return details, fmt.Errorf("failed to scan template variable - %w", err)
		}

		details.Variables = append(details.Variables, variable)
	}

	return details, nil
}

func (r *Registry) DeleteTemplate(ctx context.Context, templateId string) error {

	tx, err := r.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	// Relies on ON DELETE CASCADE constraint to delete the template variables
	_, err = tx.Exec(ctx, deleteTemplateInfo, templateId)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to delete notification template info - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("failed to commit changes - %w", err)
	}

	return nil
}

func (r *Registry) GetTemplateVariables(ctx context.Context, templateId string) ([]dto.TemplateVariable, error) {
	variables := []dto.TemplateVariable{}

	rows, err := r.conn.Query(ctx, getTemplateVariables, templateId)

	if err != nil {
		return variables, fmt.Errorf("failed to query template variables - %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var variable dto.TemplateVariable
		err := rows.Scan(
			&variable.Name,
			&variable.Type,
			&variable.Required,
			&variable.Validation,
		)

		if err != nil {
			return variables, fmt.Errorf("failed to scan template variable - %w", err)
		}

		variables = append(variables, variable)
	}

	return variables, nil
}
