package dto

type PageFilter struct {
	Page     *int `form:"page" binding:"omitempty,min=0"`
	PageSize *int `form:"pageSize" binding:"omitempty,min=0"`
}

type Page[T any] struct {
	CurrentPage  int  `json:"currentPage"`
	NextPage     *int `json:"nextPage"`
	PrevPage     *int `json:"prevPage"`
	TotalPages   int  `json:"totalPages"`
	TotalRecords int  `json:"totalRecords"`
	Data         []T  `json:"data"`
}
