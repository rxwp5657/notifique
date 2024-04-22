package dto

type PageFilter struct {
	Take *int `form:"take" binding:"omitempty,min=0"`
	Skip *int `form:"skip" binding:"omitempty,min=0"`
}
