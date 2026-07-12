package repository

import (
	"strings"
	"testing"
)

func TestBuildAttachmentOrderClause(t *testing.T) {
	if got := buildAttachmentOrderClause("file_name", "asc"); got != "file_name ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildAttachmentOrderClause("type", "DESC"); got != "type DESC" {
		t.Fatalf("got %s", got)
	}
	if got := buildAttachmentOrderClause("file_size", "asc"); got != "file_size ASC" {
		t.Fatalf("got %s", got)
	}
	if got := buildAttachmentOrderClause("bad", "nope"); got != "created_at DESC" {
		t.Fatalf("invalid should default created_at DESC, got %s", got)
	}
	evil := buildAttachmentOrderClause("id; DROP TABLE", "ASC")
	if strings.Contains(evil, "DROP") {
		t.Fatalf("raw input leaked: %s", evil)
	}
	if got := buildAttachmentOrderClause("created_at", "DeSc"); got != "created_at DESC" {
		t.Fatalf("mixed-case direction: got %s", got)
	}
}
