package storage

type notificationLog struct {
	Channel    string `dynamodbav:"channel"`
	Status     string `dynamodbav:"status"`
	StatusDate string `dynamodbav:"statusDate"`
}

type notification struct {
	Id               string            `dynamodbav:"id"`
	CreatedBy        string            `dynamodbav:"createdBy"`
	CreatedAt        string            `dynamodbav:"createdAt"`
	Title            string            `dynamodbav:"title"`
	Contents         string            `dynamodbav:"contents"`
	Image            *string           `dynamodbav:"image"`
	Topic            string            `dynamodbav:"topic"`
	Priority         string            `dynamodbav:"priority"`
	DistributionList *string           `dynamodbav:"distributionList"`
	Recipients       []string          `dynamodbav:"recipients"`
	Channels         []string          `dynamodbav:"channels"`
	Logs             []notificationLog `dynamodbav:"logs"`
}
