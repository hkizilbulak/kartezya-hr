package repository

import (
	"kartezya-hr/internal/types"
)

var attachmentSortAllowlist = map[string]bool{
	"file_name":  true,
	"type":       true,
	"file_size":  true,
	"created_at": true,
	"updated_at": true,
	"id":         true,
}

const attachmentDefaultSort = "created_at"

// buildAttachmentOrderClause maps allowlisted attachment keys to a safe ORDER BY clause.
// Default matches historical hardcoded created_at DESC.
func buildAttachmentOrderClause(sortKey, direction string) string {
	return buildAllowlistedOrder(sortKey, direction, attachmentDefaultSort, string(types.DESC), attachmentSortAllowlist)
}
