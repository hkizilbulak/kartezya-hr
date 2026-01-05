package service

import (
	"time"
)

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Data interface{} `json:"data"`
	Page PageInfo    `json:"page"`
}

// PageInfo contains pagination metadata
type PageInfo struct {
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalPages int64  `json:"total_pages"`
	Sort       string `json:"sort"`
	Direction  string `json:"direction"`
}

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}
