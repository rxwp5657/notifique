package dto

type PageFilter struct {
	NextToken  *string `form:"nextToken" binding:"omitempty"`
	MaxResults *int    `form:"maxResults" binding:"omitempty,min=1"`
}

type Page[T any] struct {
	NextToken   *string `json:"nextToken"`
	PrevToken   *string `json:"prevToken"`
	ResultCount int     `json:"resultCount"`
	Data        []T     `json:"data"`
}
