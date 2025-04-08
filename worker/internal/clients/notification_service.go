package clients

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type param string
type endpoint string

const (
	DistributionListEndpoint             endpoint = "%s/distribution-lists/%s/recipients"
	NotificationTemplateEndpoint         endpoint = "%s/notifications/templates/%s"
	NotificationStatusEndpoint           endpoint = "%s/notifications/%s/status"
	NotificationRecipientsStatusEndpoint endpoint = "%s/notifications/%s/recipients/statuses"
	UsersNotificationsEndpoint           endpoint = "%s/users/notifications"
	MaxResults                           param    = "1"
	MaxResultsParamName                  string   = "maxResults"
	NextTokenParamName                   string   = "nextToken"
	RateLimitReset                       string   = "X-RateLimit-Reset"
)

type NotificationServiceClient struct {
	AuthProvider           AuthProvider
	NotificationServiceUrl string
	NumRetries             int
	BaseDelay              time.Duration
	MaxDelay               time.Duration
}

func exponentialBackoffWithJitter(attempt int, baseDelay, maxDelay time.Duration) time.Duration {

	// Calculate the exponential delay (2^attempt * baseDelay)
	expDelay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay

	// Apply jitter (randomized delay between 50-100% of expDelay)
	jitter := expDelay / 2
	jitterInMs := jitter.Milliseconds()

	delay := min(expDelay-jitter+time.Duration(rand.Int63n(jitterInMs)), maxDelay)

	return delay
}

func (p *NotificationServiceClient) DoRequestWithBackoff(req *http.Request, retry int) (*http.Response, error) {

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return res, fmt.Errorf("error sending request - %w", err)
	}

	if res.StatusCode == http.StatusTooManyRequests && retry < p.NumRetries {
		retryAfter := res.Header.Get(RateLimitReset)
		sleepMs, err := strconv.ParseInt(retryAfter, 10, 64)

		if err != nil {
			return res, fmt.Errorf("error parsing retry-after header - %w", err)
		}

		time.Sleep(time.Duration(sleepMs+10) * time.Millisecond)
		return p.DoRequestWithBackoff(req, retry+1)
	}

	if res.StatusCode >= 500 && retry < p.NumRetries {
		sleep := exponentialBackoffWithJitter(
			retry,
			p.BaseDelay,
			p.MaxDelay)

		time.Sleep(sleep)
		return p.DoRequestWithBackoff(req, retry+1)
	}

	if res.StatusCode != http.StatusOK {
		return res, fmt.Errorf("error response from server - %s", res.Status)
	}

	return res, nil
}
