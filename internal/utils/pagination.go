package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams holds the pagination query parameters extracted from the request.
// Offset is calculated from page and limit (offset = (page - 1) * limit).
// Page is 1-indexed for the frontend; internally we work with offset for SQL.
type PaginationParams struct {
	Page   int
	Limit  int
	Offset int
}

// DefaultLimit is the default number of items per page if not specified.
const DefaultLimit = 20

// MaxLimit is the maximum number of items per page to prevent abuse.
const MaxLimit = 100

// ExtractPaginationParams extracts page and limit from query parameters.
// page defaults to 1, limit defaults to DefaultLimit (20).
// Both are clamped to reasonable values: page >= 1, 1 <= limit <= MaxLimit.
// Returns PaginationParams with page (1-indexed), limit, and offset (0-indexed).
//
// Example query: ?page=2&limit=10 returns {Page: 2, Limit: 10, Offset: 10}
// Example query: ?page=invalid gives page defaults to 1
// Example query: ?limit=1000 clamps limit to MaxLimit (100)
func ExtractPaginationParams(c *gin.Context) PaginationParams {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", strconv.Itoa(DefaultLimit))

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	// Offset is 0-indexed for SQL OFFSET clause
	offset := (page - 1) * limit

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}

// BuildPaginationMeta returns standard pagination metadata for list responses.
// total_pages is rounded up (e.g. total=21, limit=20 => total_pages=2).
func BuildPaginationMeta(total int, pag PaginationParams) map[string]any {
	totalPages := 0
	if total > 0 {
		totalPages = (total + pag.Limit - 1) / pag.Limit
	}

	return map[string]any{
		"total":       total,
		"page":        pag.Page,
		"limit":       pag.Limit,
		"total_pages": totalPages,
		"has_next":    pag.Page < totalPages,
		"has_prev":    pag.Page > 1,
	}
}
