package storage

type distributionList struct {
	Name      string `db:"name"`
	Recipient string `db:"recipient"`
}

type distributionListSummary struct {
	Name               string
	NumberOfRecipients string
}

type distributionListKey struct {
	Name string `json:"name"`
}

const INSERT_DISTRIBUTION_LIST_RECIPIENT = `
INSERT INTO distribution_lists(
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
	distribution_lists
WHERE
	"name" = @name
GROUP BY
	"name";
`

const GET_DISTRIBUTION_LISTS = `
SELECT
	"name",
	COUNT(*) AS num_recipients
FROM
	distribution_lists
%s
GROUP BY
	"name"
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

const DELETE_DISTRIBUTION_LIST_RECIPIENTS = `
DELETE FROM
	distribution_lists
WHERE
	"name" = @name AND
	recipient ANY (@recipients);
`

const GET_DISTRIBUTION_LIST_RECIPIENTS = `
SELECT
	recipient
FROM
	distribution_lists
WHERE
	%s
ORDER BY
	recipient
LIMIT
	@limit;
`
