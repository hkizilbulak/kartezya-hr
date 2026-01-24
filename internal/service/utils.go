package service

import (
	"fmt"
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

// parseDate parses date strings in multiple formats (ISO 8601 and YYYY-MM-DD)
func parseDate(dateStr string) (*time.Time, error) {
	if dateStr == "" {
		return nil, nil
	}

	// Try ISO 8601 format first (e.g., 2011-07-12T00:00:00.000Z)
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return &t, nil
	}

	// Try RFC3339Nano format
	if t, err := time.Parse(time.RFC3339Nano, dateStr); err == nil {
		return &t, nil
	}

	// Try YYYY-MM-DD format
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return &t, nil
	}

	// If none of the formats work, return error
	return nil, fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD or ISO 8601)", dateStr)
}
