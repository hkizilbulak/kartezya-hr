package database

import (
	"strings"
	"testing"

	"kartezya-hr/internal/domain"
)

func TestSyncTableIDSequenceSQL_UsesMaxID(t *testing.T) {
	table := domain.GetTableName("hr_employee_grades")
	if table == "" {
		t.Fatal("empty table name")
	}
	sql := buildSyncTableIDSequenceSQL(table)
	for _, want := range []string{"setval", "pg_get_serial_sequence", "MAX(id)", table} {
		if !strings.Contains(sql, want) {
			t.Fatalf("sql missing %q: %s", want, sql)
		}
	}
}
