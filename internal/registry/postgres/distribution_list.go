package postgresresgistry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/notifique/internal"
	"github.com/notifique/internal/dto"
	"github.com/notifique/internal/registry"
)

type distributionList struct {
	Name      string `db:"name"`
	Recipient string `db:"recipient"`
}

type recipient struct {
	Recipient string `db:"recipient"`
}

type distributionListSummary struct {
	Name               string `db:"name"`
	NumberOfRecipients int    `db:"num_recipients"`
}

type distributionListKey struct {
	Name string `json:"name"`
}

const InsertDistributionList = `
INSERT INTO distribution_lists (
	"name",
	num_recipients
) VALUES (
	@name,
	@numRecipients
);
`

const InsertDistributionListRecipient = `
INSERT INTO distribution_list_recipients(
	"name",
	recipient
) VALUES (
	@name,
	@recipient
) ON CONFLICT
	("name", "recipient")
  DO NOTHING;
`

const GetDistributionList = `
SELECT
	"name",
	COUNT(*) AS num_recipients
FROM
	distribution_list_recipients
WHERE
	"name" = @name
GROUP BY
	"name";
`

const GetDistributionLists = `
SELECT
	*
FROM
	distribution_lists
%s
ORDER BY
	"name"
LIMIT
	@limit;
`

const DeleteDistributionList = `
DELETE FROM
	distribution_lists
WHERE
	"name" = @name;
`

const DeleteAllRecipientsOfDistributionList = `
DELETE FROM
	distribution_list_recipients
WHERE
	"name" = @name;
`

const DeleteDistributionListRecipients = `
DELETE FROM
	distribution_list_recipients
WHERE
	"name" = @name AND
	recipient = ANY (@recipients);
`

const GetDistributionListRecipients = `
SELECT
	recipient
FROM
	distribution_list_recipients
WHERE
	%s
ORDER BY
	recipient
LIMIT
	@limit;
`

const UpdateRecipientsCount = `
UPDATE
	distribution_lists
SET
	num_recipients = @numRecipients
WHERE
	"name" = @name;
`

func getDistributionListSummary(ctx context.Context, listName string, rQuerier RowQuerier) (*dto.DistributionListSummary, error) {

	args := pgx.NamedArgs{"name": listName}

	var summary dto.DistributionListSummary

	err := rQuerier.QueryRow(ctx, GetDistributionList, args).Scan(
		&summary.Name,
		&summary.NumberOfRecipients,
	)

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get distribution list - %w", err)
	}

	return &summary, nil
}

func (ps *Registry) CreateDistributionList(ctx context.Context, distributionList dto.DistributionList) error {

	list, err := getDistributionListSummary(ctx, distributionList.Name, ps.conn)

	if err != nil {
		return err
	}

	if list != nil {
		return internal.DistributionListAlreadyExists{
			Name: list.Name,
		}
	}

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"name":          distributionList.Name,
		"numRecipients": len(distributionList.Recipients),
	}

	_, err = tx.Exec(ctx, InsertDistributionList, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to create distribution list - %w", err)
	}

	recipientsArgs := make([]pgx.NamedArgs, 0, len(distributionList.Recipients))

	for _, recipient := range distributionList.Recipients {
		recipientsArgs = append(recipientsArgs, pgx.NamedArgs{
			"name":      distributionList.Name,
			"recipient": recipient,
		})
	}

	err = batchInsert(
		ctx,
		InsertDistributionListRecipient,
		recipientsArgs,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to insert distribution list recipients - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("commit failed - %w", err)
	}

	return nil
}

func (ps *Registry) GetDistributionLists(ctx context.Context, filters dto.PageFilter) (dto.Page[dto.DistributionListSummary], error) {

	page := dto.Page[dto.DistributionListSummary]{}

	args := pgx.NamedArgs{"limit": internal.PageSize}

	nextTokenFilter := ""

	if filters.MaxResults != nil {
		limit := *filters.MaxResults
		args["limit"] = limit
	}

	if filters.NextToken != nil {
		nextTokenFilter = `WHERE ("name") > (@name)`

		var unmarsalledKey distributionListKey
		err := registry.UnmarshalKey(*filters.NextToken, &unmarsalledKey)

		if err != nil {
			return page, err
		}

		args["name"] = unmarsalledKey.Name
	}

	query := fmt.Sprintf(GetDistributionLists, nextTokenFilter)
	rows, err := ps.conn.Query(ctx, query, args)

	if err != nil {
		return page, fmt.Errorf("failed to query rows - %w", err)
	}

	defer rows.Close()

	summaries, err := pgx.CollectRows(rows, pgx.RowToStructByName[distributionListSummary])

	if err != nil {
		return page, fmt.Errorf("failed to collect rows - %w", err)
	}

	numSummaries := len(summaries)

	if numSummaries == args["limit"] {
		lastSummary := summaries[numSummaries-1]

		lastSummaryKey := distributionListKey{
			Name: lastSummary.Name,
		}

		key, err := registry.MarshalKey(lastSummaryKey)

		if err != nil {
			return page, err
		}

		page.NextToken = &key
	}

	dtoSummaries := make([]dto.DistributionListSummary, 0, len(summaries))

	for _, summary := range summaries {
		s := dto.DistributionListSummary{
			Name:               summary.Name,
			NumberOfRecipients: summary.NumberOfRecipients,
		}

		dtoSummaries = append(dtoSummaries, s)
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(dtoSummaries)
	page.Data = dtoSummaries

	return page, nil
}

func (ps *Registry) DeleteDistributionList(ctx context.Context, distlistName string) error {

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{"name": distlistName}

	// Should also delete its recipients
	_, err = tx.Exec(ctx, DeleteDistributionList, args)

	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to delete distribution list - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return fmt.Errorf("commit failed - %w", err)
	}

	return nil
}

func (ps *Registry) GetRecipients(ctx context.Context, distlistName string, filters dto.PageFilter) (dto.Page[string], error) {

	page := dto.Page[string]{}

	summary, err := getDistributionListSummary(ctx, distlistName, ps.conn)

	if err != nil {
		return page, fmt.Errorf("failed to get summary - %w", err)
	}

	if summary == nil {
		return page, internal.EntityNotFound{
			Id:   distlistName,
			Type: registry.DistributionListType,
		}
	}

	args := pgx.NamedArgs{"limit": internal.PageSize}
	whereFilters := make([]string, 0)

	if filters.MaxResults != nil {
		limit := *filters.MaxResults
		args["limit"] = limit
	}

	if filters.NextToken != nil {
		filter := `("name", recipient) > (@name, @recipient)`
		whereFilters = append(whereFilters, filter)

		var unmarsalledKey distributionList
		err := registry.UnmarshalKey(*filters.NextToken, &unmarsalledKey)

		if err != nil {
			return page, err
		}

		if unmarsalledKey.Name != distlistName {
			return page, fmt.Errorf("invalid key %s", *filters.NextToken)
		}

		args["name"] = unmarsalledKey.Name
		args["recipient"] = unmarsalledKey.Recipient
	} else {
		filter := "name = @name"
		whereFilters = append(whereFilters, filter)
		args["name"] = distlistName
	}

	whereStmt := strings.Join(whereFilters, "AND")
	query := fmt.Sprintf(GetDistributionListRecipients, whereStmt)

	rows, err := ps.conn.Query(ctx, query, args)

	if err != nil {
		return page, fmt.Errorf("failed to query rows - %w", err)
	}

	defer rows.Close()

	recipients, err := pgx.CollectRows(rows, pgx.RowToStructByName[recipient])

	if err != nil {
		return page, fmt.Errorf("failed to collect rows - %w", err)
	}

	numRecipients := len(recipients)

	if numRecipients == args["limit"] {
		lastRecipient := recipients[numRecipients-1]

		dl := distributionList{
			Name:      distlistName,
			Recipient: lastRecipient.Recipient,
		}

		key, err := registry.MarshalKey(dl)

		if err != nil {
			return page, err
		}

		page.NextToken = &key
	}

	recipientsNames := make([]string, 0, len(recipients))

	for _, r := range recipients {
		recipientsNames = append(recipientsNames, r.Recipient)
	}

	page.PrevToken = filters.NextToken
	page.ResultCount = len(recipients)
	page.Data = recipientsNames

	return page, nil
}

func (ps *Registry) AddRecipients(ctx context.Context, distlistName string, recipients []string) (*dto.DistributionListSummary, error) {

	exists, err := getDistributionListSummary(ctx, distlistName, ps.conn)

	if err != nil {
		return nil, fmt.Errorf("failed to get summary - %w", err)
	}

	if exists == nil {
		return nil, internal.EntityNotFound{
			Id:   distlistName,
			Type: registry.DistributionListType,
		}
	}

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start transaction - %w", err)
	}

	recipientsArgs := make([]pgx.NamedArgs, 0, len(recipients))

	for _, recipient := range recipients {
		recipientsArgs = append(recipientsArgs, pgx.NamedArgs{
			"name":      distlistName,
			"recipient": recipient,
		})
	}

	err = batchInsert(
		ctx,
		InsertDistributionListRecipient,
		recipientsArgs,
		tx,
	)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed add recipients - %w", err)
	}

	summary, err := getDistributionListSummary(ctx, distlistName, tx)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to get summary - %w", err)
	}

	countArgs := pgx.NamedArgs{
		"numRecipients": summary.NumberOfRecipients,
		"name":          distlistName,
	}

	_, err = tx.Exec(ctx, UpdateRecipientsCount, countArgs)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to update recipients count - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("commit failed - %w", err)
	}

	return summary, nil
}

func (ps *Registry) DeleteRecipients(ctx context.Context, distlistName string, recipients []string) (*dto.DistributionListSummary, error) {

	exists, err := getDistributionListSummary(ctx, distlistName, ps.conn)

	if err != nil {
		return nil, fmt.Errorf("failed to get summary - %w", err)
	}

	if exists == nil {
		return nil, internal.EntityNotFound{
			Id:   distlistName,
			Type: registry.DistributionListType,
		}
	}

	tx, err := ps.conn.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start transaction - %w", err)
	}

	args := pgx.NamedArgs{
		"name":       distlistName,
		"recipients": recipients,
	}

	_, err = tx.Exec(ctx, DeleteDistributionListRecipients, args)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed delete recipients - %w", err)
	}

	summary, err := getDistributionListSummary(ctx, distlistName, tx)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to get summary - %w", err)
	}

	countArgs := pgx.NamedArgs{
		"numRecipients": summary.NumberOfRecipients,
		"name":          distlistName,
	}

	_, err = tx.Exec(ctx, UpdateRecipientsCount, countArgs)

	if err != nil {
		tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to update recipients count - %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to commit delete recipients - %w", err)
	}

	return summary, nil
}
