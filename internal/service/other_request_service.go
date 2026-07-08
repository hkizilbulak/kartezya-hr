package service

import (
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"time"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"kartezya-hr/internal/types"
)

type OtherRequestService interface {
	CreateRequestType(reqType *domain.RequestType, userID uint) error
	GetRequestTypeByID(id uint) (*domain.RequestType, error)
	GetAllRequestTypes(limit, offset int, sortParams types.SortParams) ([]*domain.RequestType, int64, error)
	UpdateRequestType(reqType *domain.RequestType, userID uint) error
	DeleteRequestType(id uint, userID uint) error

	CreateRequest(req *domain.OtherRequest, userID uint) error
	GetRequestByID(id uint) (*domain.OtherRequest, error)
	GetRequestsByUserID(userID uint) ([]*domain.OtherRequest, error)
	GetAllRequests(filterEmployeeID *uint, limit, offset int, sortParams types.SortParams) ([]*domain.OtherRequest, int64, error)
	UpdateRequest(req *domain.OtherRequest, userID uint, isAdmin bool) error
	CancelRequest(id uint, userID uint, isAdmin bool) error
	CompleteRequest(id uint, completerID uint) error
	RollbackRequest(id uint, userID uint) error

	UploadRequestDocument(requestID uint, file *multipart.FileHeader, userID uint, isAdmin bool, isHR bool) (*domain.Attachment, error)
	GetRequestDocuments(requestID uint, userID uint, isAdmin bool, isHR bool) ([]domain.Attachment, error)
	DeleteRequestDocument(documentID string, userID uint, isAdmin bool, isHR bool) error
	DownloadRequestDocument(documentID string, userID uint, isAdmin bool, isHR bool) (string, error)
}

type otherRequestService struct {
	repo              repository.OtherRequestRepository
	attachmentRepo    repository.AttachmentRepository
	auditService      AuditService
	emailService      EmailService
	mailConfigService MailConfigService
	storage           StorageProvider
	employeeRepo      repository.EmployeeRepository
}

func NewOtherRequestService(
	repo repository.OtherRequestRepository,
	attachmentRepo repository.AttachmentRepository,
	auditService AuditService,
	emailService EmailService,
	storage StorageProvider,
	employeeRepo repository.EmployeeRepository,
	mailConfigService MailConfigService,
) OtherRequestService {
	return &otherRequestService{
		repo:              repo,
		attachmentRepo:    attachmentRepo,
		auditService:      auditService,
		emailService:      emailService,
		mailConfigService: mailConfigService,
		storage:           storage,
		employeeRepo:      employeeRepo,
	}
}

// ==================== TALEP TÜRÜ İŞLEMLERİ ====================

func (s *otherRequestService) CreateRequestType(reqType *domain.RequestType, userID uint) error {
	return s.repo.CreateRequestType(reqType, fmt.Sprintf("%d", userID))
}

func (s *otherRequestService) GetRequestTypeByID(id uint) (*domain.RequestType, error) {
	return s.repo.GetRequestTypeByID(id)
}

func (s *otherRequestService) GetAllRequestTypes(limit, offset int, sortParams types.SortParams) ([]*domain.RequestType, int64, error) {
	return s.repo.GetAllRequestTypes(limit, offset, sortParams)
}

func (s *otherRequestService) UpdateRequestType(reqType *domain.RequestType, userID uint) error {
	return s.repo.UpdateRequestType(reqType, fmt.Sprintf("%d", userID))
}

func (s *otherRequestService) DeleteRequestType(id uint, userID uint) error {
	return s.repo.DeleteRequestType(id, fmt.Sprintf("%d", userID))
}

// ==================== TALEP İŞLEMLERİ ====================

func (s *otherRequestService) CreateRequest(req *domain.OtherRequest, userID uint) error {
	req.Status = domain.RequestStatusActive
	err := s.repo.CreateRequest(req, userID, fmt.Sprintf("%d", userID))

	if err == nil {
		s.auditService.CreateAuditLog("OtherRequest", req.ID, "CREATE", nil, req, fmt.Sprintf("%d", userID))

		go func(r *domain.OtherRequest) {
			if r.Employee == nil {
				log.Printf("[OtherRequest] employee info missing, skipping email")
				return
			}

			variables := map[string]interface{}{
				"fullname":    fmt.Sprintf("%s %s", r.Employee.FirstName, r.Employee.LastName),
				"requestType": r.RequestType.Name,
			}

			to, cc, bcc, templateCode, cfgErr := s.mailConfigService.ResolveRecipients("INFO_EMAIL_NEW_OTHER_DEMAND")
			if cfgErr != nil || len(to) == 0 {
				log.Printf("[OtherRequest] INFO_EMAIL_NEW_OTHER_DEMAND config not found, falling back to SendNewRequestEmail: %v", cfgErr)
				if emailErr := s.emailService.SendNewRequestEmail(r); emailErr != nil {
					log.Printf("[OtherRequest] E-POSTA GÖNDERİM HATASI (fallback): %v", emailErr)
				}
				return
			}

			if templateCode == "" {
				templateCode = "new-request-email"
			}

			if emailErr := s.emailService.SendTemplateEmailWithCC(to, cc, bcc, "Yeni Talep Oluşturuldu", templateCode, variables); emailErr != nil {
				log.Printf("[OtherRequest] E-POSTA GÖNDERİM HATASI: %v", emailErr)
			}
		}(req)
	}
	return err
}

func (s *otherRequestService) GetRequestByID(id uint) (*domain.OtherRequest, error) {
	return s.repo.GetRequestByID(id)
}

func (s *otherRequestService) GetRequestsByUserID(userID uint) ([]*domain.OtherRequest, error) {
	return s.repo.GetRequestsByUserID(userID)
}

func (s *otherRequestService) GetAllRequests(filterEmployeeID *uint, limit, offset int, sortParams types.SortParams) ([]*domain.OtherRequest, int64, error) {
	return s.repo.GetAllRequests(filterEmployeeID, limit, offset, sortParams)
}

func (s *otherRequestService) UpdateRequest(req *domain.OtherRequest, userID uint, isAdmin bool) error {
	existing, err := s.repo.GetRequestByID(req.ID)
	if err != nil {
		return err
	}
	employee, err := s.employeeRepo.GetByUserID(userID)
	if err != nil {
		return errors.New("çalışan kaydı bulunamadı")
	}
	if !isAdmin && existing.EmployeeID != employee.ID {
		return errors.New("sadece kendi taleplerinizi güncelleyebilirsiniz")
	}
	if existing.Status == domain.RequestStatusCompleted {
		return errors.New("Tamamlanmış talepler güncellenemez")
	}
	req.Status = domain.RequestStatusActive
	err = s.repo.UpdateRequest(req, fmt.Sprintf("%d", userID))
	if err == nil {
		s.auditService.CreateAuditLog("OtherRequest", req.ID, "UPDATE", existing, req, fmt.Sprintf("%d", userID))
	}
	return err
}

func (s *otherRequestService) CancelRequest(id uint, userID uint, isAdmin bool) error {
	req, err := s.repo.GetRequestByID(id)
	if err != nil {
		return err
	}
	if req.Status == domain.RequestStatusCompleted && !isAdmin {
		return errors.New("tamamlanmış talepler silinemez")
	}
	req.Status = domain.RequestStatusCancelled
	err = s.repo.UpdateRequest(req, fmt.Sprintf("%d", userID))
	if err == nil {
		s.auditService.CreateAuditLog("OtherRequest", id, "CANCEL", nil, req, fmt.Sprintf("%d", userID))
	}
	return err
}

func (s *otherRequestService) CompleteRequest(id uint, completerID uint) error {
	req, err := s.repo.GetRequestByID(id)
	if err != nil {
		return err
	}

	req.Status = domain.RequestStatusCompleted
	req.CompletedBy = &completerID
	now := time.Now()
	req.CompletedAt = &now

	err = s.repo.UpdateRequest(req, fmt.Sprintf("%d", completerID))

	if err == nil {
		s.auditService.CreateAuditLog("OtherRequest", id, "COMPLETE", nil, req, fmt.Sprintf("%d", completerID))

		go func(r *domain.OtherRequest) {
			if r.Employee == nil {
				log.Printf("[OtherRequest] employee info missing, skipping completed email")
				return
			}

			employeeEmail := r.Employee.CompanyEmail
			if employeeEmail == "" {
				employeeEmail = r.Employee.Email
			}

			variables := map[string]interface{}{
				"fullname":    fmt.Sprintf("%s %s", r.Employee.FirstName, r.Employee.LastName),
				"requestType": r.RequestType.Name,
			}

			// INFO_EMAIL_OTHER_DEMAND_COMPLETED config'inden CC/BCC çöz
			_, cc, bcc, templateCode, cfgErr := s.mailConfigService.ResolveRecipients("INFO_EMAIL_OTHER_DEMAND_COMPLETED")
			if cfgErr != nil {
				log.Printf("[OtherRequest] INFO_EMAIL_OTHER_DEMAND_COMPLETED config not found, sending without CC/BCC: %v", cfgErr)
			}
			if templateCode == "" {
				templateCode = "request-completed-email"
			}

			if emailErr := s.emailService.SendTemplateEmailWithCC(
				[]string{employeeEmail}, cc, bcc,
				"Talebiniz Tamamlandı", templateCode, variables,
			); emailErr != nil {
				log.Printf("[OtherRequest] E-POSTA GÖNDERİM HATASI (Complete): %v", emailErr)
			}
		}(req)
	}
	return err
}

func (s *otherRequestService) RollbackRequest(id uint, userID uint) error {
	req, err := s.repo.GetRequestByID(id)
	if err != nil {
		return err
	}
	req.Status = domain.RequestStatusActive
	req.CompletedBy = nil
	req.CompletedAt = nil
	err = s.repo.UpdateRequest(req, fmt.Sprintf("%d", userID))
	if err == nil {
		s.auditService.CreateAuditLog("OtherRequest", id, "ROLLBACK", nil, req, fmt.Sprintf("%d", userID))
	}
	return err
}

// ==================== DÖKÜMAN YÖNETİMİ ====================

func (s *otherRequestService) UploadRequestDocument(requestID uint, file *multipart.FileHeader, userID uint, isAdmin bool, isHR bool) (*domain.Attachment, error) {
	request, err := s.repo.GetRequestByID(requestID)
	if err != nil {
		return nil, errors.New("talep bulunamadı")
	}
	if request.Status == domain.RequestStatusCompleted {
		return nil, errors.New("tamamlanmış taleplere doküman yüklenemez")
	}
	employee, err := s.employeeRepo.GetByUserID(userID)
	if !isAdmin && !isHR {
		if err != nil {
			return nil, errors.New("çalışan kaydı bulunamadı")
		}
		if request.EmployeeID != employee.ID {
			return nil, errors.New("sadece kendi taleplerinize doküman yükleyebilirsiniz")
		}
	}
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("dosya açılamadı: %w", err)
	}
	defer src.Close()
	timestamp := time.Now().Format("20060102150405")
	docID := fmt.Sprintf("%d_%s", requestID, timestamp)
	storagePath := GenerateStoragePath(domain.AttachmentRelatedTypeOtherRequest, file.Filename, docID)
	if err := s.storage.Upload(src, storagePath); err != nil {
		return nil, fmt.Errorf("dosya file server'a yüklenemedi: %w", err)
	}
	attachment := &domain.Attachment{
		ID:          domain.GenerateUUID(),
		OwnerID:     userID,
		RelatedType: domain.AttachmentRelatedTypeOtherRequest,
		RelatedID:   &requestID,
		Type:        domain.AttachmentTypeDocument,
		Status:      domain.AttachmentStatusLinked,
		FileName:    file.Filename,
		Path:        storagePath,
		ContentType: file.Header.Get("Content-Type"),
		FileSize:    file.Size,
	}
	if err := s.attachmentRepo.Create(attachment); err != nil {
		_ = s.storage.Delete(storagePath)
		return nil, fmt.Errorf("döküman kaydı oluşturulamadı: %w", err)
	}
	s.auditService.CreateAuditLog("Attachment", 0, "UPLOAD", nil, attachment, fmt.Sprintf("%d", userID))
	return attachment, nil
}

func (s *otherRequestService) GetRequestDocuments(requestID uint, userID uint, isAdmin bool, isHR bool) ([]domain.Attachment, error) {
	if !isAdmin && !isHR {
		req, err := s.repo.GetRequestByID(requestID)
		if err != nil {
			return nil, err
		}
		emp, err := s.employeeRepo.GetByUserID(userID)
		if err != nil || req.EmployeeID != emp.ID {
			return nil, errors.New("yetkisiz erişim")
		}
	}
	return s.attachmentRepo.FindByRelatedRecord(domain.AttachmentRelatedTypeOtherRequest, requestID)
}

func (s *otherRequestService) DeleteRequestDocument(documentID string, userID uint, isAdmin bool, isHR bool) error {
	attachment, err := s.attachmentRepo.FindByID(documentID)
	if err != nil {
		return errors.New("döküman bulunamadı")
	}
	if attachment.RelatedType != domain.AttachmentRelatedTypeOtherRequest {
		return errors.New("geçersiz talep dökümanı")
	}
	request, err := s.repo.GetRequestByID(*attachment.RelatedID)
	if err != nil {
		return errors.New("talep bulunamadı")
	}
	if request.Status == domain.RequestStatusCompleted {
		return errors.New("tamamlanmış taleplerden doküman silinemez")
	}
	if !isAdmin && !isHR {
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil {
			return errors.New("çalışan kaydı bulunamadı")
		}
		if request.EmployeeID != employee.ID {
			return errors.New("sadece kendi taleplerinizdeki dokümanları silebilirsiniz")
		}
	}
	if err := s.storage.Delete(attachment.Path); err != nil {
		fmt.Printf("Storage hatası: %v\n", err)
	}
	err = s.attachmentRepo.DeleteAttachment(documentID)
	if err == nil {
		s.auditService.CreateAuditLog("Attachment", 0, "DELETE", attachment, nil, fmt.Sprintf("%d", userID))
	}
	return err
}

func (s *otherRequestService) DownloadRequestDocument(documentID string, userID uint, isAdmin bool, isHR bool) (string, error) {
	attachment, err := s.attachmentRepo.FindByID(documentID)
	if err != nil {
		return "", errors.New("döküman bulunamadı")
	}
	if !isAdmin && !isHR {
		request, err := s.repo.GetRequestByID(*attachment.RelatedID)
		if err != nil {
			return "", err
		}
		employee, err := s.employeeRepo.GetByUserID(userID)
		if err != nil || request.EmployeeID != employee.ID {
			return "", errors.New("yetkisiz erişim")
		}
	}
	url, err := s.storage.GeneratePresignedURL(attachment.Path, 15)
	if err != nil {
		return "", fmt.Errorf("indirme linki üretilemedi: %w", err)
	}
	return url, nil
}
