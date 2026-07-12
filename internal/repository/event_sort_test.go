package repository

import (
	"strings"
	"testing"

	"kartezya-hr/internal/types"
)

func TestEventSortKeyMapping(t *testing.T) {
	for _, key := range []string{"name", "start_date", "end_date", "location", "audience_filter", "status", "type"} {
		got := buildEventOrderClause(key, "ASC")
		if !strings.Contains(got, key+" ASC") {
			t.Fatalf("expected %q in clause, got %q", key, got)
		}
	}
	if got := buildEventOrderClause("not_a_column", "DESC"); !strings.Contains(got, "start_date DESC") {
		t.Fatalf("invalid key should default to start_date, got %q", got)
	}
	if dir := types.NormalizeSortDirection("asc", "DESC"); dir != "ASC" {
		t.Fatalf("expected ASC, got %s", dir)
	}
	if strings.Contains(buildEventOrderClause("'; DROP", "ASC"), "DROP") {
		t.Fatal("raw input must not pass through")
	}
}

func TestSanitizeSortRequestTypes(t *testing.T) {
	allowed := map[string]bool{
		"created_at": true, "name": true, "id": true, "description": true, "active": true,
	}
	if got := types.AllowedSortOrDefault("description", allowed, "created_at"); got != "description" {
		t.Fatalf("expected description, got %q", got)
	}
	if got := types.AllowedSortOrDefault("active", allowed, "created_at"); got != "active" {
		t.Fatalf("expected active, got %q", got)
	}
	if got := types.AllowedSortOrDefault("status", allowed, "created_at"); got != "created_at" {
		t.Fatalf("status is not a request-type column; expected created_at default, got %q", got)
	}
}
