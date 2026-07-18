package repository

import (
	"os"
	"strings"
	"testing"
)

func TestNormalizedLikePatternForCityFilter(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{name: "empty string skipped", in: "", want: ""},
		{name: "whitespace skipped", in: "   ", want: ""},
		{name: "null skipped", in: "null", want: ""},
		{name: "undefined skipped", in: "undefined", want: ""},
		{name: "partial match pattern", in: "izmi", want: "%izmi%"},
		{name: "trimmed partial", in: "  izmi  ", want: "%izmi%"},
		{name: "preserves case for LOWER in SQL", in: "IZMI", want: "%IZMI%"},
		{name: "full city name", in: "İzmir", want: "%İzmir%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizedLikePattern(tt.in); got != tt.want {
				t.Fatalf("normalizedLikePattern(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Regression: pagination total uses GetTotalCountWithFilters, which must apply the
// same city LIKE filter as GetAllWithFilters (list + internal count).
func TestGetTotalCountWithFiltersSourceIncludesCityFilter(t *testing.T) {
	src, err := os.ReadFile("employee_repository.go")
	if err != nil {
		t.Fatalf("read employee_repository.go: %v", err)
	}
	content := string(src)

	const marker = "func (r *employeeRepository) GetTotalCountWithFilters"
	start := strings.Index(content, marker)
	if start < 0 {
		t.Fatal("GetTotalCountWithFilters not found")
	}
	rest := content[start:]
	// Next method after GetTotalCountWithFilters
	next := strings.Index(rest[len(marker):], "\nfunc (r *employeeRepository)")
	if next < 0 {
		t.Fatal("could not bound GetTotalCountWithFilters body")
	}
	body := rest[:len(marker)+next]

	requiredSnippets := []string{
		`filters["city"]`,
		`normalizedLikePattern(city)`,
		`LOWER(%s.city) LIKE LOWER(?)`,
	}
	for _, snip := range requiredSnippets {
		if !strings.Contains(body, snip) {
			t.Fatalf("GetTotalCountWithFilters missing city filter snippet %q", snip)
		}
	}

	// Same city WHERE shape must also exist in GetAllWithFilters
	listMarker := "func (r *employeeRepository) GetAllWithFilters"
	listStart := strings.Index(content, listMarker)
	if listStart < 0 {
		t.Fatal("GetAllWithFilters not found")
	}
	listRest := content[listStart:]
	listNext := strings.Index(listRest[len(listMarker):], "\nfunc (r *employeeRepository)")
	if listNext < 0 {
		t.Fatal("could not bound GetAllWithFilters body")
	}
	listBody := listRest[:len(listMarker)+listNext]
	for _, snip := range requiredSnippets {
		if !strings.Contains(listBody, snip) {
			t.Fatalf("GetAllWithFilters missing city filter snippet %q", snip)
		}
	}
}
