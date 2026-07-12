package types

import "testing"

func TestNormalizeSortDirection(t *testing.T) {
	tests := []struct {
		name       string
		direction  string
		defaultDir string
		want       string
	}{
		{name: "asc lower", direction: "asc", defaultDir: "ASC", want: "ASC"},
		{name: "ASC upper", direction: "ASC", defaultDir: "DESC", want: "ASC"},
		{name: "desc lower", direction: "desc", defaultDir: "ASC", want: "DESC"},
		{name: "DESC upper", direction: "DESC", defaultDir: "ASC", want: "DESC"},
		{name: "mixed Desc", direction: "Desc", defaultDir: "ASC", want: "DESC"},
		{name: "mixed aSc", direction: "aSc", defaultDir: "DESC", want: "ASC"},
		{name: "empty uses default ASC", direction: "", defaultDir: "ASC", want: "ASC"},
		{name: "empty uses default DESC", direction: "", defaultDir: "DESC", want: "DESC"},
		{name: "invalid uses default ASC", direction: "sideways", defaultDir: "ASC", want: "ASC"},
		{name: "invalid uses default DESC", direction: "nope", defaultDir: "desc", want: "DESC"},
		{name: "whitespace", direction: "  desc  ", defaultDir: "ASC", want: "DESC"},
		{name: "invalid default falls back ASC", direction: "bad", defaultDir: "weird", want: "ASC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSortDirection(tt.direction, tt.defaultDir)
			if got != tt.want {
				t.Fatalf("NormalizeSortDirection(%q, %q) = %q, want %q", tt.direction, tt.defaultDir, got, tt.want)
			}
		})
	}
}

func TestAllowedSortOrDefault(t *testing.T) {
	allowed := map[string]bool{
		"employee_name": true,
		"company_name":  true,
		"created_at":    true,
	}

	if got := AllowedSortOrDefault("company_name", allowed, "employee_name"); got != "company_name" {
		t.Fatalf("expected company_name, got %q", got)
	}
	if got := AllowedSortOrDefault("employee_id", allowed, "employee_name"); got != "employee_name" {
		t.Fatalf("invalid key should fall back to default, got %q", got)
	}
	if got := AllowedSortOrDefault("", allowed, "created_at"); got != "created_at" {
		t.Fatalf("empty key should fall back to default, got %q", got)
	}
}
