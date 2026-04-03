package pagination

import (
	"net/http"
	"strconv"
)

// ParseQuery reads limit/offset query params from an HTTP request and applies validation defaults.
func ParseQuery(r *http.Request) PaginationRequest {
	q := r.URL.Query()
	limit, _ := strconv.ParseInt(q.Get("limit"), 10, 32)
	offset, _ := strconv.ParseInt(q.Get("offset"), 10, 32)
	req := PaginationRequest{
		Limit:  int32(limit),
		Offset: int32(offset),
	}
	req.Validate()
	return req
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Limit  int32 `json:"limit" example:"20"`
	Offset int32 `json:"offset" example:"0"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Total  int64 `json:"total" example:"100"` // Total number of records
	Limit  int32 `json:"limit" example:"20"`  // Max items per page
	Offset int32 `json:"offset" example:"0"`  // Number of items to skip
}

// SearchRequest combines search query with pagination
type SearchRequest struct {
	Query string `json:"query,omitempty" example:"search term"`
	PaginationRequest
}

// Validate validates and applies defaults to pagination parameters
func (p *PaginationRequest) Validate() {
	if p.Limit <= 0 || p.Limit > 100 {
		p.Limit = 20
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}

// NewPaginationResponse creates a pagination response
func NewPaginationResponse(total int64, limit, offset int32) PaginationResponse {
	return PaginationResponse{
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

// HasNextPage returns true if there are more pages
func (p *PaginationResponse) HasNextPage() bool {
	return int64(p.Offset+p.Limit) < p.Total
}

// HasPrevPage returns true if there are previous pages
func (p *PaginationResponse) HasPrevPage() bool {
	return p.Offset > 0
}

// TotalPages returns total number of pages
func (p *PaginationResponse) TotalPages() int64 {
	if p.Limit == 0 {
		return 0
	}
	return (p.Total + int64(p.Limit) - 1) / int64(p.Limit)
}

// CurrentPage returns current page number (1-indexed)
func (p *PaginationResponse) CurrentPage() int64 {
	if p.Limit == 0 {
		return 0
	}
	return int64(p.Offset)/int64(p.Limit) + 1
}
