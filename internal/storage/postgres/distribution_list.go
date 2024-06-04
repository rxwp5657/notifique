package storage

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

const INSERT_DISTRIBUTION_LIST = `
INSERT INTO distribution_lists (
	"name",
	num_recipients
) VALUES (
	@name,
	@numRecipients
);
`

const INSERT_DISTRIBUTION_LIST_RECIPIENT = `
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

const GET_DISTRIBUTION_LIST = `
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

const GET_DISTRIBUTION_LISTS = `
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

const DELETE_DISTRIBUTION_LIST = `
DELETE FROM
	distribution_lists
WHERE
	"name" = @name;
`

const DELETE_ALL_RECIPIENTS_DISTRIBUTION_LIST = `
DELETE FROM
	distribution_list_recipients
WHERE
	"name" = @name;
`

const DELETE_DISTRIBUTION_LIST_RECIPIENTS = `
DELETE FROM
	distribution_list_recipients
WHERE
	"name" = @name AND
	recipient = ANY (@recipients);
`

const GET_DISTRIBUTION_LIST_RECIPIENTS = `
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

const UPDATE_RECIPIENTS_COUNT = `
UPDATE
	distribution_lists
SET
	num_recipients = @numRecipients
WHERE
	"name" = @name;
`
