package servers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/notifique/shared/dto"
)

func applyPaginationParams[T any](originalData []T, nextToken string, maxResults string) (dto.Page[T], error) {

	data := make([]T, len(originalData))
	copy(data, originalData)

	page := dto.Page[T]{
		NextToken:   nil,
		PrevToken:   nil,
		ResultCount: len(data),
		Data:        data,
	}

	if nextToken != "" {
		idx, err := strconv.Atoi(nextToken)
		page.PrevToken = &nextToken

		if err != nil {
			return page, fmt.Errorf("error parsing nextToken - %w", err)
		}

		if idx >= len(data) {
			page.Data = []T{}
			page.ResultCount = 0
			data = []T{}
		} else {
			page.Data = data[idx:]
			page.ResultCount = len(data) - idx
			data = data[idx:]
		}
	}

	if maxResults != "" {
		maxResultsInt, err := strconv.Atoi(maxResults)
		page.NextToken = &maxResults

		if err != nil || maxResultsInt <= 0 {
			return page, fmt.Errorf("error parsing maxResults - %w", err)
		}

		if maxResultsInt >= len(data) {
			maxResultsInt = len(data)
			page.NextToken = nil
		}

		page.Data = data[:maxResultsInt]
		page.ResultCount = maxResultsInt
	}

	return page, nil
}

func MakeDistributionListHandler(responses map[string]any) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("/distribution-lists/%s/recipients", r.PathValue("id"))

		response, ok := responses[key]

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		respRecipients, ok := response.([]string)

		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		nextToken := r.URL.Query().Get("nextToken")
		maxResults := r.URL.Query().Get("maxResults")

		page, err := applyPaginationParams(respRecipients, nextToken, maxResults)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		marshalledResponse, err := json.Marshal(page)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(marshalledResponse)
	}
}

func MakeNotificationTemplateHandler(responses map[string]any) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("/notifications/templates/%s", r.PathValue("id"))

		response, ok := responses[key]

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		templateDetails, ok := response.(dto.NotificationTemplateDetails)

		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		marshalledResponse, err := json.Marshal(templateDetails)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(marshalledResponse)
	}
}

func MakeNotificationStatusHandler(responses map[string]any) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("/notifications/%s/recipients/statuses", r.PathValue("id"))

		response, ok := responses[key]

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		respStatuses, ok := response.([]dto.RecipientNotificationStatus)

		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		nextToken := r.URL.Query().Get("nextToken")
		maxResults := r.URL.Query().Get("maxResults")
		channels := r.URL.Query()["channels"]

		statuses := []dto.RecipientNotificationStatus{}

		if len(channels) > 0 {
			channelsSet := make(map[string]struct{}, len(channels))
			for _, channel := range channels {
				channelsSet[channel] = struct{}{}
			}
			for _, status := range respStatuses {
				if _, ok := channelsSet[status.Channel]; ok {
					statuses = append(statuses, status)
				}
			}
		} else {
			statuses = make([]dto.RecipientNotificationStatus, len(respStatuses))
			copy(statuses, respStatuses)
		}

		page, err := applyPaginationParams(statuses, nextToken, maxResults)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		marshalledResponse, err := json.Marshal(page)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(marshalledResponse)
	}
}

func MakeUserNotificationsHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func NewTestServer(method string, handlerFunc func(w http.ResponseWriter, r *http.Request)) *httptest.Server {

	handler := http.NewServeMux()

	handler.HandleFunc(method, handlerFunc)

	return httptest.NewServer(handler)
}
