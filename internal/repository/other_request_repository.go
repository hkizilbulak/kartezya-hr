package repository

import (
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/types"
	"log"

	"gorm.io/gorm"
)

type OtherRequestRepository interface {
	// Talep Türü
	CreateRequestType(reqType *domain.RequestType, createdBy string) error
	GetRequestTypeByID(id uint) (*domain.RequestType, error)
	GetAllRequestTypes(limit, offset int, sortParams types.SortParams) ([]*domain.RequestType, int64, error)
	UpdateRequestType(reqType *domain.RequestType, modifiedBy string) error
	DeleteRequestType(id uint, deletedBy string) error

	// Talep
	CreateRequest(req *domain.OtherRequest, userID uint, createdBy string) error
	GetRequestByID(id uint) (*domain.OtherRequest, error)
	GetRequestsByUserID(userID uint) ([]*domain.OtherRequest, error)
	GetAllRequests(filterEmployeeID *uint, limit, offset int, sortParams types.SortParams) ([]*domain.OtherRequest, int64, error)
	UpdateRequest(req *domain.OtherRequest, modifiedBy string) error
	CancelRequest(id uint, deletedBy string) error
}

type otherRequestRepository struct {
	db *gorm.DB
}

func NewOtherRequestRepository(db *gorm.DB) OtherRequestRepository {
	return &otherRequestRepository{db: db}
}

// ==================== TALEP TÜRÜ ====================

func (r *otherRequestRepository) CreateRequestType(reqType *domain.RequestType, createdBy string) error {
	reqType.CreatedBy = createdBy
	reqType.ModifiedBy = createdBy
	return r.db.Create(reqType).Error
}

func (r *otherRequestRepository) GetRequestTypeByID(id uint) (*domain.RequestType, error) {
	var reqType domain.RequestType
	err := r.db.Where("id = ? AND deleted = ?", id, false).First(&reqType).Error
	return &reqType, err
}

func (r *otherRequestRepository) GetAllRequestTypes(limit, offset int, sortParams types.SortParams) ([]*domain.RequestType, int64, error) {
	var reqTypes []*domain.RequestType
	var total int64

	query := r.db.Model(&domain.RequestType{}).Where("deleted = ?", false)
	query.Count(&total)

	allowedSort := map[string]bool{
		"created_at":  true,
		"name":        true,
		"id":          true,
		"description": true,
		"active":      true,
	}
	safeSort := types.AllowedSortOrDefault(sortParams.Sort, allowedSort, "created_at")
	orderDir := types.NormalizeSortDirection(sortParams.Direction, "ASC")
	query = query.Order(safeSort + " " + orderDir)

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err := query.Find(&reqTypes).Error
	return reqTypes, total, err
}

func (r *otherRequestRepository) UpdateRequestType(reqType *domain.RequestType, modifiedBy string) error {
	return r.db.Model(reqType).Updates(map[string]interface{}{
		"name":        reqType.Name,
		"description": reqType.Description,
		"active":      reqType.Active,
		"modified_by": modifiedBy,
	}).Error
}

func (r *otherRequestRepository) DeleteRequestType(id uint, deletedBy string) error {
	return r.db.Model(&domain.RequestType{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted":     true,
		"modified_by": deletedBy,
	}).Error
}

// ==================== TALEP ====================

func (r *otherRequestRepository) CreateRequest(req *domain.OtherRequest, userID uint, createdBy string) error {
	var employee domain.Employee
	if err := r.db.Where("user_id = ? AND deleted = ?", userID, false).First(&employee).Error; err != nil {
		return fmt.Errorf("Bu kullanıcıya ait aktif bir çalışan kaydı bulunamadı!")
	}

	req.EmployeeID = employee.ID
	req.CreatedBy = createdBy
	req.ModifiedBy = createdBy

	if err := r.db.Create(req).Error; err != nil {
		return err
	}

	return r.db.Preload("Employee").Preload("RequestType").First(req, req.ID).Error
}

func (r *otherRequestRepository) GetRequestByID(id uint) (*domain.OtherRequest, error) {
	var req domain.OtherRequest
	err := r.db.Preload("Employee").
		Preload("RequestType").
		Preload("Completer").
		Preload("Attachments").
		Where("id = ? AND deleted = ?", id, false).First(&req).Error
	return &req, err
}

func (r *otherRequestRepository) GetRequestsByUserID(userID uint) ([]*domain.OtherRequest, error) {
	var reqs []*domain.OtherRequest
	var emp domain.Employee

	if err := r.db.Where("user_id = ? AND deleted = ?", userID, false).First(&emp).Error; err != nil {
		return nil, fmt.Errorf("Kullanıcı profilinize tanımlı aktif bir çalışan kaydı bulunamadı!")
	}

	err := r.db.Preload("Employee").
		Preload("RequestType").
		Preload("Attachments").
		Where("employee_id = ? AND deleted = ?", emp.ID, false).
		Order("created_at DESC").
		Find(&reqs).Error

	return reqs, err
}

func (r *otherRequestRepository) GetAllRequests(filterEmployeeID *uint, limit, offset int, sortParams types.SortParams) ([]*domain.OtherRequest, int64, error) {
	var reqs []*domain.OtherRequest
	var total int64

	reqTable := domain.GetTableName("hr_other_requests")

	query := r.db.Model(&domain.OtherRequest{}).
		Preload("Employee").
		Preload("RequestType").
		Preload("Attachments").
		Where(fmt.Sprintf("%s.deleted = ?", reqTable), false)

	if filterEmployeeID != nil {
		query = query.Where(fmt.Sprintf("%s.employee_id = ?", reqTable), *filterEmployeeID)
	}

	if err := query.Count(&total).Error; err != nil {
		log.Printf("ERROR: Count query failed: %v", err)
		return nil, 0, err
	}

	orderClause, needsEmployeeJoin, needsTypeJoin := buildOtherRequestOrderClause(sortParams.Sort, sortParams.Direction)
	if needsEmployeeJoin {
		empTable := domain.GetTableName("hr_employees")
		query = query.Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.employee_id", empTable, empTable, reqTable))
	}
	if needsTypeJoin {
		typeTable := domain.GetTableName("hr_request_types")
		query = query.Joins(fmt.Sprintf("LEFT JOIN %s ON %s.id = %s.request_type_id", typeTable, typeTable, reqTable))
	}
	query = query.Order(orderClause)

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	err := query.Find(&reqs).Error
	if err != nil {
		log.Printf("ERROR: Find query failed: %v", err)
		return nil, 0, err
	}

	log.Printf("DEBUG: GetAllRequests - Bulunan: %d, Toplam: %d", len(reqs), total)
	return reqs, total, nil
}

func (r *otherRequestRepository) UpdateRequest(req *domain.OtherRequest, modifiedBy string) error {
	return r.db.Model(&domain.OtherRequest{}).
		Where("id = ?", req.ID).
		Updates(map[string]interface{}{
			"request_type_id": req.RequestTypeID,
			"description":     req.Description,
			"status":          req.Status,
			"completed_by":    req.CompletedBy,
			"completed_at":    req.CompletedAt,
			"modified_by":     modifiedBy,
		}).Error
}

func (r *otherRequestRepository) CancelRequest(id uint, deletedBy string) error {
	return r.db.Model(&domain.OtherRequest{}).
		Where("id = ? AND deleted = ?", id, false).
		Updates(map[string]interface{}{
			"status":      domain.RequestStatusCancelled,
			"modified_by": deletedBy,
		}).Error
}
