package repository

import (
    "kartezya-hr/internal/domain"
    "kartezya-hr/internal/types"
    "time"

    "gorm.io/gorm"
)

// HolidayRepository interface for holiday operations
type HolidayRepository interface {
    Create(holiday *domain.Holiday) error
    GetByID(id uint) (*domain.Holiday, error)
    GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Holiday, int64, error)
    GetByDateRange(startDate, endDate time.Time) ([]*domain.Holiday, error)
    Update(holiday *domain.Holiday) error
    Delete(id uint) error
    GetHolidaysBetweenDates(startDate, endDate time.Time) ([]*domain.Holiday, error)
}

// holidayRepository implements HolidayRepository
type holidayRepository struct {
    db *gorm.DB
}

// NewHolidayRepository creates a new holiday repository
func NewHolidayRepository(db *gorm.DB) HolidayRepository {
    return &holidayRepository{db: db}
}

// Create creates a new holiday
func (r *holidayRepository) Create(holiday *domain.Holiday) error {
    return r.db.Create(holiday).Error
}

// GetByID gets a holiday by ID
func (r *holidayRepository) GetByID(id uint) (*domain.Holiday, error) {
    var holiday domain.Holiday
    err := r.db.First(&holiday, id).Error
    if err != nil {
        return nil, err
    }
    return &holiday, nil
}

// GetAll gets all holidays with pagination
func (r *holidayRepository) GetAll(limit, offset int, sortParams types.SortParams) ([]*domain.Holiday, int64, error) {
    var holidays []*domain.Holiday
    var total int64

    // Count total records
    countQuery := r.db.Model(&domain.Holiday{}).Where("deleted = ?", false)
    if err := countQuery.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // Build main query
    query := r.db.Where("deleted = ?", false)

    // Apply sorting
    if sortParams.Sort != "" {
        orderClause := sortParams.Sort
        if sortParams.Direction == "DESC" {
            orderClause += " DESC"
        } else {
            orderClause += " ASC"
        }
        query = query.Order(orderClause)
    } else {
        query = query.Order("holiday_date ASC")
    }

    // Apply pagination
    if limit > 0 {
        query = query.Limit(limit)
    }
    if offset > 0 {
        query = query.Offset(offset)
    }

    err := query.Find(&holidays).Error
    return holidays, total, err
}

// GetByDateRange gets holidays between two dates
func (r *holidayRepository) GetByDateRange(startDate, endDate time.Time) ([]*domain.Holiday, error) {
    var holidays []*domain.Holiday
    err := r.db.Where("holiday_date >= ? AND holiday_date <= ? AND deleted = ?", startDate, endDate, false).
        Order("holiday_date ASC").
        Find(&holidays).Error
    return holidays, err
}

// GetHolidaysBetweenDates gets holidays between two dates (alias for GetByDateRange)
func (r *holidayRepository) GetHolidaysBetweenDates(startDate, endDate time.Time) ([]*domain.Holiday, error) {
    return r.GetByDateRange(startDate, endDate)
}

// Update updates a holiday
func (r *holidayRepository) Update(holiday *domain.Holiday) error {
    return r.db.Save(holiday).Error
}

// Delete soft deletes a holiday
func (r *holidayRepository) Delete(id uint) error {
    return r.db.Model(&domain.Holiday{}).Where("id = ?", id).Update("deleted", true).Error
}
