package dto

import "math"

type Pagination struct {
	CurrentPage int
	TotalPages  int
	TotalItems  int
	PageSize    int
	HasPrev     bool
	PrevPage    int
	HasNext     bool
	NextPage    int
	BaseURL     string
	Target      string
	ExtraParams string
}

func NewPagination(page, pageSize, total int, baseURL, target, extraParams string) *Pagination {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 30
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages == 0 {
		totalPages = 1
	}

	if page > totalPages {
		page = totalPages
	}

	return &Pagination{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalItems:  total,
		PageSize:    pageSize,
		HasPrev:     page > 1,
		PrevPage:    page - 1,
		HasNext:     page < totalPages,
		NextPage:    page + 1,
		BaseURL:     baseURL,
		Target:      target,
		ExtraParams: extraParams,
	}
}
