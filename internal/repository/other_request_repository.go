package repository

import (
    "fmt"
    "kartezya-hr/internal/domain"
    "kartezya-hr/internal/types"

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
    GetAllRequests(limit, offset int, sortParams types.SortParams) ([]*domain.OtherRequest, int64, error)
    UpdateRequest(req *domain.OtherRequest, modifiedBy string) error
    CancelRequest(id uint, deletedBy string) error

    // Döküman Yönetimi
    DeleteAttachment(documentID string) error
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
    if err != nil {
        return nil, err
    }
    return &reqType, nil
}

func (r *otherRequestRepository) GetAllRequestTypes(limit, offset int, sortParams types.SortParams) ([]*domain.RequestType, int64, error) {
    var reqTypes []*domain.RequestType
    var total int64

    sortField := "created_at"
    direction := "DESC"

    if sortParams.Direction == "ASC" {
        direction = "ASC"
    }

    if sortParams.Sort != "" {
        switch sortParams.Sort {
        case "name":
            sortField = "name"
        case "active":
            sortField = "active"
        case "created_at":
            sortField = "created_at"
        }
    }

    query := r.db.Model(&domain.RequestType{}).Where("deleted = ?", false)
    query.Count(&total)

    err := query.Order(fmt.Sprintf("%s %s", sortField, direction)).
        Limit(limit).Offset(offset).
        Find(&reqTypes).Error

    return reqTypes, total, err
}

func (r *otherRequestRepository) UpdateRequestType(reqType *domain.RequestType, modifiedBy string) error {
    reqType.ModifiedBy = modifiedBy
    return r.db.Where("deleted = ?", false).Save(reqType).Error
}

func (r *otherRequestRepository) DeleteRequestType(id uint, deletedBy string) error {
    return r.db.Unscoped().Where("id = ?", id).Delete(&domain.RequestType{}).Error
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

    // Servis katmanında mail atılırken Employee ve RequestType bilgilerinin boş kalmaması için ilişkileri yüklüyoruz
    return r.db.Preload("Employee").Preload("RequestType").First(req, req.ID).Error
}

func (r *otherRequestRepository) GetRequestByID(id uint) (*domain.OtherRequest, error) {
    var req domain.OtherRequest
    err := r.db.Preload("Employee").
        Preload("RequestType").
        Preload("Completer").
        Preload("Attachments").
        Where("id = ? AND deleted = ?", id, false).First(&req).Error
    if err != nil {
        return nil, err
    }
    return &req, nil
}

func (r *otherRequestRepository) GetAllRequests(limit, offset int, sortParams types.SortParams) ([]*domain.OtherRequest, int64, error) {
    var reqs []*domain.OtherRequest
    var total int64

    sortField := "created_at"
    direction := "DESC"

    if sortParams.Direction == "ASC" {
        direction = "ASC"
    }

    if sortParams.Sort != "" {
        switch sortParams.Sort {
        case "status":
            sortField = "status"
        case "created_at":
            sortField = "created_at"
        case "updated_at":
            sortField = "updated_at"
        }
    }

    query := r.db.Model(&domain.OtherRequest{}).Where("deleted = ?", false)
    query.Count(&total)

    err := query.Preload("Employee").
        Preload("RequestType").
        Preload("Attachments").
        Order(fmt.Sprintf("%s %s", sortField, direction)).
        Limit(limit).Offset(offset).
        Find(&reqs).Error

    return reqs, total, err
}

func (r *otherRequestRepository) UpdateRequest(req *domain.OtherRequest, modifiedBy string) error {
    req.ModifiedBy = modifiedBy
    return r.db.Where("deleted = ?", false).Save(req).Error
}

// Talep silindiğinde silme yapılmaz, durumu CANCELLED yapılır.
func (r *otherRequestRepository) CancelRequest(id uint, deletedBy string) error {
    return r.db.Model(&domain.OtherRequest{}).
        Where("id = ? AND deleted = ?", id, false).
        Updates(map[string]interface{}{
            "status":      domain.RequestStatusCancelled,
            "modified_by": deletedBy,
        }).Error
}

// DeleteAttachment dökümanı veritabanından tamamen siler.
func (r *otherRequestRepository) DeleteAttachment(documentID string) error {
    return r.db.Where("id = ?", documentID).Delete(&domain.Attachment{}).Error
}