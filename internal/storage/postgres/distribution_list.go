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
