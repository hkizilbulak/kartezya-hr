package database

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"kartezya-hr/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	closedGradeBoundsPattern = regexp.MustCompile(`^\s*L[0-9]+\s*\(\s*([0-9]+)\s*-\s*([0-9]+)\s*\)\s*$`)
	openGradeBoundsPattern   = regexp.MustCompile(`^\s*L[0-9]+\s*\(\s*([0-9]+)\s*\+\s*\)\s*$`)
)

// BackfillMissingGradeBounds fills only NULL min_year/max_year values on
// non-deleted grades. Open-ended ranges intentionally keep max_year NULL.
func BackfillMissingGradeBounds(db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("database is nil")
	}

	var updated int64
	err := db.Transaction(func(tx *gorm.DB) error {
		var grades []domain.Grade
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("deleted = ? AND (min_year IS NULL OR max_year IS NULL)", false).
			Order("id ASC").
			Find(&grades).Error; err != nil {
			return fmt.Errorf("failed to load grades with missing bounds: %w", err)
		}

		for i := range grades {
			grade := &grades[i]
			minYear, maxYear, err := parseGradeBounds(grade.Name)
			if err != nil {
				return fmt.Errorf("grade id=%d name=%q has invalid bounds: %w", grade.ID, grade.Name, err)
			}

			updates := make(map[string]interface{}, 2)
			if grade.MinYear == nil {
				updates["min_year"] = minYear
			}
			if grade.MaxYear == nil && maxYear != nil {
				updates["max_year"] = *maxYear
			}
			if len(updates) == 0 {
				continue
			}

			result := tx.Model(&domain.Grade{}).
				Where("id = ? AND deleted = ?", grade.ID, false).
				UpdateColumns(updates)
			if result.Error != nil {
				return fmt.Errorf("failed to backfill grade id=%d: %w", grade.ID, result.Error)
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("failed to backfill grade id=%d: row changed concurrently", grade.ID)
			}
			updated += result.RowsAffected
		}

		return nil
	})
	if err != nil {
		return 0, err
	}
	return updated, nil
}

func parseGradeBounds(name string) (int, *int, error) {
	if matches := closedGradeBoundsPattern.FindStringSubmatch(name); matches != nil {
		minYear, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, nil, fmt.Errorf("invalid minimum year: %w", err)
		}
		maxYear, err := strconv.Atoi(matches[2])
		if err != nil {
			return 0, nil, fmt.Errorf("invalid maximum year: %w", err)
		}
		if maxYear <= minYear {
			return 0, nil, fmt.Errorf("maximum year must be greater than minimum year")
		}
		return minYear, &maxYear, nil
	}

	if matches := openGradeBoundsPattern.FindStringSubmatch(name); matches != nil {
		minYear, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, nil, fmt.Errorf("invalid minimum year: %w", err)
		}
		return minYear, nil, nil
	}

	return 0, nil, fmt.Errorf("expected name format like L3(4-6) or L6(10+), got %q", strings.TrimSpace(name))
}
