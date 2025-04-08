package testutils

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/notifique/service/internal/dto"
	sdto "github.com/notifique/shared/dto"
)

func makePageURLQuery(req *http.Request, filters sdto.PageFilter) url.Values {
	q := req.URL.Query()

	if filters.NextToken != nil {
		q.Add("nextToken", fmt.Sprint(*filters.NextToken))
	}

	if filters.MaxResults != nil {
		q.Add("maxResults", fmt.Sprint(*filters.MaxResults))
	}

	return q
}

func AddPaginationFilters(req *http.Request, filters *sdto.PageFilter) {

	if req == nil || filters == nil {
		return
	}

	q := makePageURLQuery(req, *filters)

	req.URL.RawQuery = q.Encode()
}

func AddUserFilters(req *http.Request, filters *dto.UserNotificationFilters) {

	if req == nil || filters == nil {
		return
	}

	q := makePageURLQuery(req, filters.PageFilter)

	for _, t := range filters.Topics {
		q.Add("topics", t)
	}

	req.URL.RawQuery = q.Encode()
}

func AddNotificationTemplateFilters(req *http.Request, filters *dto.NotificationTemplateFilters) {

	if req == nil || filters == nil {
		return
	}

	q := makePageURLQuery(req, filters.PageFilter)

	if filters.TemplateName != nil {
		q.Add("templateName", *filters.TemplateName)
	}

	req.URL.RawQuery = q.Encode()
}

func Echo[T any](data T) T {
	return data
}
