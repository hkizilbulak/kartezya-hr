package service

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
)

func TestGradeReportRowScansCurrentGradeID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var row types.GradeReportRow
	if err := db.Raw(`SELECT CAST(30 AS INTEGER) AS current_grade_id`).Scan(&row).Error; err != nil {
		t.Fatalf("scan current_grade_id: %v", err)
	}
	if row.CurrentGradeID == nil || *row.CurrentGradeID != 30 {
		t.Fatalf("current_grade_id = %v, want 30", row.CurrentGradeID)
	}
}

func TestExpectedGradeTransitionSoonBoundaries(t *testing.T) {
	professionStartDate := time.Date(2019, 8, 9, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name           string
		date           time.Time
		transitionYear int
		want           bool
	}{
		{name: "seven years L4 more than nine months", date: time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC), transitionYear: 8, want: false},
		{name: "seven years three months L4 exactly nine months", date: time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC), transitionYear: 8, want: true},
		{name: "seven years six months L4 less than nine months", date: time.Date(2027, 2, 9, 0, 0, 0, 0, time.UTC), transitionYear: 8, want: true},
		{name: "six years eleven months L4 thirteen months", date: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC), transitionYear: 8, want: false},
		{name: "five years two months L3 ten months", date: time.Date(2024, 10, 9, 0, 0, 0, 0, time.UTC), transitionYear: 6, want: false},
		{name: "five years three months L3 exactly nine months", date: time.Date(2024, 11, 9, 0, 0, 0, 0, time.UTC), transitionYear: 6, want: true},
		{name: "five years six months L3 less than nine months", date: time.Date(2025, 2, 9, 0, 0, 0, 0, time.UTC), transitionYear: 6, want: true},
		{name: "four years L2 boundary reached", date: time.Date(2023, 8, 9, 0, 0, 0, 0, time.UTC), transitionYear: 4, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := expectedGradeTransitionSoon(professionStartDate, test.date, test.transitionYear); got != test.want {
				t.Fatalf("expectedGradeTransitionSoon() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestApplyExpectedGradesLiveBoundaries(t *testing.T) {
	min2, max4, min4, max6, min6, max8, min8, max10 := 2, 4, 4, 6, 6, 8, 8, 10
	grades := []*domain.Grade{
		{AuditableModel: domain.AuditableModel{ID: 50}, Name: "L5", MinYear: &min8, MaxYear: &max10},
		{AuditableModel: domain.AuditableModel{ID: 20}, Name: "L2", MinYear: &min2, MaxYear: &max4},
		{AuditableModel: domain.AuditableModel{ID: 40}, Name: "L4", MinYear: &min6, MaxYear: &max8},
		{AuditableModel: domain.AuditableModel{ID: 30}, Name: "L3", MinYear: &min4, MaxYear: &max6},
	}
	asOfDate := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	l4ID, l5ID := int64(40), int64(50)
	professionStartL2 := "2022-08-09"
	professionStartL4 := "2016-08-09"
	rows := []types.GradeReportRow{
		{CurrentGrade: "L2", ProfessionStartDate: &professionStartL2},
		{CurrentGradeID: &l4ID, CurrentGrade: "L4", ProfessionStartDate: &professionStartL4},
		{CurrentGradeID: &l5ID, CurrentGrade: "L5", ProfessionStartDate: &professionStartL4},
		{CurrentGrade: "L2", ProfessionStartDate: nil},
	}

	if err := applyExpectedGrades(rows, grades, asOfDate); err != nil {
		t.Fatalf("applyExpectedGrades: %v", err)
	}

	if rows[0].ExpectedGrade != "L3" {
		t.Fatalf("name fallback next grade = %q, want L3", rows[0].ExpectedGrade)
	}
	if rows[1].ExpectedGrade != "L5" {
		t.Fatalf("ID lookup next grade = %q, want L5", rows[1].ExpectedGrade)
	}
	if rows[2].ExpectedGrade != "L5" {
		t.Fatalf("top grade fallback = %q, want L5", rows[2].ExpectedGrade)
	}
	if rows[3].ExpectedGrade != "L2" {
		t.Fatalf("missing profession start date fallback = %q, want L2", rows[3].ExpectedGrade)
	}
}

func TestResolveCurrentReportGradeNormalizesDefensiveLabelFallback(t *testing.T) {
	min4, max6 := 4, 6
	grade := &domain.Grade{
		AuditableModel: domain.AuditableModel{ID: 30},
		Name:           "L3",
		MinYear:        &min4,
		MaxYear:        &max6,
	}
	row := &types.GradeReportRow{CurrentGrade: "L3(4-6)"}
	got := resolveCurrentReportGrade(
		row,
		map[int64]*domain.Grade{30: grade},
		map[string]*domain.Grade{normalizeGradeLabel(grade.Name): grade},
	)
	if got != grade {
		t.Fatalf("normalized fallback did not resolve: %#v", got)
	}
}

func TestApplyExpectedGradesRejectsUnresolvedCurrentGrade(t *testing.T) {
	professionStartDate := "2021-02-01"
	rows := []types.GradeReportRow{{
		ID:                  99,
		CurrentGrade:        "UNKNOWN(4-6)",
		ProfessionStartDate: &professionStartDate,
	}}
	err := applyExpectedGrades(rows, nil, time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("unresolved current grade must return an error")
	}
}

func TestGetGradeReportDataLiveExamples(t *testing.T) {
	min2, max4, min4, max6, min6, max8, min8, max10 := 2, 4, 4, 6, 6, 8, 8, 10
	grades := []*domain.Grade{
		{AuditableModel: domain.AuditableModel{ID: 50}, Name: "L5(8-10)", MinYear: &min8, MaxYear: &max10},
		{AuditableModel: domain.AuditableModel{ID: 20}, Name: "L2(2-4)", MinYear: &min2, MaxYear: &max4},
		{AuditableModel: domain.AuditableModel{ID: 40}, Name: "L4(6-8)", MinYear: &min6, MaxYear: &max8},
		{AuditableModel: domain.AuditableModel{ID: 30}, Name: "L3(4-6)", MinYear: &min4, MaxYear: &max6},
	}
	asOfDate := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	starts := []string{
		"2021-02-01",
		"2022-08-01",
		"2019-09-01",
		"2018-06-01",
		"2022-10-01",
	}
	currentGrades := []string{"L3(4-6)", "L2(2-4)", "L4(6-8)", "L5(8-10)", "L2(2-4)"}
	currentGradeIDs := []int64{30, 20, 40, 50, 20}
	want := []string{"L4(6-8)", "L3(4-6)", "L4(6-8)", "L5(8-10)", "L3(4-6)"}
	rows := make([]types.GradeReportRow, len(starts))
	for index := range starts {
		rows[index] = types.GradeReportRow{
			CurrentGrade:        currentGrades[index],
			CurrentGradeID:      &currentGradeIDs[index],
			ProfessionStartDate: &starts[index],
		}
	}

	svc := &reportService{
		employeeRepo: &stubEGEmployeeRepo{gradeReportRows: rows},
		gradeRepo:    &stubEGGradeRepo{grades: grades},
		now:          func() time.Time { return asOfDate },
	}
	report, err := svc.GetGradeReportData(&types.GradeReportFilter{})
	if err != nil {
		t.Fatalf("GetGradeReportData: %v", err)
	}
	for index := range want {
		if report.Rows[index].ExpectedGrade != want[index] {
			t.Errorf("case %d expected_grade = %q, want %q", index+1, report.Rows[index].ExpectedGrade, want[index])
		}
	}
}
