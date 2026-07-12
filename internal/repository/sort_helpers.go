package repository

import (
	"fmt"
	"kartezya-hr/internal/types"
)

// buildAllowlistedOrder returns a safe "column DIR" ORDER BY fragment.
// sortKey is never interpolated unless it is in allowed.
func buildAllowlistedOrder(sortKey, direction, defaultKey, defaultDir string, allowed map[string]bool) string {
	direction = types.NormalizeSortDirection(direction, defaultDir)
	key := types.AllowedSortOrDefault(sortKey, allowed, defaultKey)
	return fmt.Sprintf("%s %s", key, direction)
}
