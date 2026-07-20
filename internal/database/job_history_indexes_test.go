package database

import (
	"strings"
	"testing"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
)

func TestResolveJobHistoryTableName_UsesConfiguredPrefix(t *testing.T) {
	cases := []struct {
		prefix string
		want   string
	}{
		{prefix: "hr", want: "hr_job_history"},
		{prefix: "hr_test", want: "hr_test_job_history"},
		{prefix: "hr_prod", want: "hr_prod_job_history"},
	}

	for _, tc := range cases {
		cfg := &config.Config{}
		cfg.Database.TablePrefix = tc.prefix
		domain.SetConfig(cfg)

		got := ResolveJobHistoryTableName()
		if got != tc.want {
			t.Fatalf("prefix=%s: got %q, want %q", tc.prefix, got, tc.want)
		}
	}
}

func TestBuildCreateJobHistoryReferenceDateUniqueIndexSQL_IsPrefixAwareAndIdempotent(t *testing.T) {
	sqlProd := BuildCreateJobHistoryReferenceDateUniqueIndexSQL("hr_job_history")
	sqlTest := BuildCreateJobHistoryReferenceDateUniqueIndexSQL("hr_test_job_history")

	for _, part := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS",
		`"ux_hr_job_history_job_id_reference_date_active"`,
		`"hr_job_history"`,
		"(job_id, reference_date)",
		"reference_date IS NOT NULL",
		"status IN ('RUNNING', 'SUCCESS')",
	} {
		if !strings.Contains(sqlProd, part) {
			t.Fatalf("prod DDL missing %q: %s", part, sqlProd)
		}
	}

	for _, part := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS",
		`"ux_hr_test_job_history_job_id_reference_date_active"`,
		`"hr_test_job_history"`,
	} {
		if !strings.Contains(sqlTest, part) {
			t.Fatalf("test DDL missing %q: %s", part, sqlTest)
		}
	}

	if sqlProd == sqlTest {
		t.Fatal("test and prod DDL must differ by resolved table/index names")
	}

	// Second generation is identical → safe for repeated startup.
	sqlProd2 := BuildCreateJobHistoryReferenceDateUniqueIndexSQL("hr_job_history")
	if sqlProd != sqlProd2 {
		t.Fatal("DDL generation must be deterministic/idempotent")
	}
}

func TestBuildJobHistoryReferenceDateUniqueIndexName(t *testing.T) {
	got := BuildJobHistoryReferenceDateUniqueIndexName("hr_test_job_history")
	want := "ux_hr_test_job_history_job_id_reference_date_active"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestQuoteIdent(t *testing.T) {
	if quoteIdent(`hr_job_history`) != `"hr_job_history"` {
		t.Fatal("unexpected quote")
	}
	if quoteIdent(`weird"name`) != `"weird""name"` {
		t.Fatal("unexpected escaped quote")
	}
}
