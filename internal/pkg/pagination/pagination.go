package pagination

import (
	"net/http"
	"strconv"

	"github.com/bangun-ekosistem/service-api/internal/domain"
)

type Params struct {
	Page    int
	PerPage int
}

func ParseFromRequest(r *http.Request) Params {
	p := Params{
		Page:    1,
		PerPage: 20,
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if v, err := strconv.Atoi(page); err == nil && v > 0 {
			p.Page = v
		}
	}

	if perPage := r.URL.Query().Get("per_page"); perPage != "" {
		if v, err := strconv.Atoi(perPage); err == nil && v > 0 {
			p.PerPage = v
		}
	}

	if p.PerPage > 100 {
		p.PerPage = 100
	}

	return p
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.PerPage
}

func (p Params) ToMeta(total int) *domain.PaginationMeta {
	totalPages := total / p.PerPage
	if total%p.PerPage > 0 {
		totalPages++
	}
	return &domain.PaginationMeta{
		Page:       p.Page,
		PerPage:    p.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}
}
