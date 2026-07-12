package types

import "strings"

// SortDirection represents the direction for sorting
type SortDirection string

const (
	ASC  SortDirection = "ASC"
	DESC SortDirection = "DESC"
)

// SortParams represents sorting parameters
type SortParams struct {
	Sort      string `json:"sort" form:"sort" query:"sort"`
	Direction string `json:"direction" form:"direction" query:"direction"`
}

// NormalizeSortDirection accepts asc/desc in any common casing and returns ASC or DESC.
// Invalid or empty values return defaultDir when it is ASC/DESC; otherwise ASC.
func NormalizeSortDirection(direction, defaultDir string) string {
	switch strings.ToUpper(strings.TrimSpace(direction)) {
	case string(ASC):
		return string(ASC)
	case string(DESC):
		return string(DESC)
	default:
		switch strings.ToUpper(strings.TrimSpace(defaultDir)) {
		case string(DESC):
			return string(DESC)
		default:
			return string(ASC)
		}
	}
}

// AllowedSortOrDefault returns sortKey when it is in allowed; otherwise defaultKey.
// Use this so invalid keys fall back to the endpoint's explicit default, not an unrelated column.
func AllowedSortOrDefault(sortKey string, allowed map[string]bool, defaultKey string) string {
	if allowed[sortKey] {
		return sortKey
	}
	return defaultKey
}
