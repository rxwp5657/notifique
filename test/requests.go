package test

import (
	"fmt"
	"net/http"

	"github.com/notifique/dto"
)

func AddPaginationFilters(req *http.Request, filters *dto.PageFilter) {

	if req == nil || filters == nil {
		return
	}

	q := req.URL.Query()

	if filters.NextToken != nil {
		q.Add("nextToken", fmt.Sprint(*filters.NextToken))
	}

	if filters.MaxResults != nil {
		q.Add("maxResults", fmt.Sprint(*filters.MaxResults))
	}

	req.URL.RawQuery = q.Encode()
}
