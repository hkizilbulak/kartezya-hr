package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
	"time"

	"gorm.io/gorm"
)

type SaveKvkkConsentDto struct {
	Action            string
	PhotoConsent      string
	KvkkText          string
	PrivacyPolicy     string
	AntiBriberyPolicy string
}

type SettingsService interface {
	GetSettings(userID uint) (*domain.UserSetting, error)
	SaveKvkkConsent(userID uint, req SaveKvkkConsentDto, clientIP string, userAgent string) (*domain.UserSetting, error)
}

type settingsService struct {
	repo         repository.SettingsRepository
	auditService AuditService
}

func NewSettingsService(repo repository.SettingsRepository, auditService AuditService) SettingsService {
	return &settingsService{
		repo:         repo,
		auditService: auditService,
	}
}

func (s *settingsService) GetSettings(userID uint) (*domain.UserSetting, error) {
	setting, err := s.repo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create default settings if not exists
			defaultSetting := &domain.UserSetting{
				UserID:                userID,
				PhotoConsent:          "PENDING",
				KvkkText:              "PENDING",
				PrivacyPolicy:         "PENDING",
				AntiBriberyPolicy:     "PENDING",
				KvkkStatus:            "PENDING",
				KvkkApproved:          false,
				KvkkApprovedAt:        nil,
				KvkkRejectedAt:        nil,
				KvkkLastPostponedAt:   nil,
				PromotionEmailAllowed: true,
				PromotionSmsAllowed:   true,
			}
			defaultSetting.CreatedBy = "SYSTEM"
			defaultSetting.ModifiedBy = "SYSTEM"
			if err := s.repo.Create(defaultSetting); err != nil {
				return nil, err
			}
			return defaultSetting, nil
		}
		return nil, err
	}
	return setting, nil
}

func (s *settingsService) SaveKvkkConsent(userID uint, req SaveKvkkConsentDto, clientIP string, userAgent string) (*domain.UserSetting, error) {
	if req.Action != "SUBMIT" && req.Action != "REMIND_LATER" {
		return nil, errors.New("invalid action, must be SUBMIT or REMIND_LATER")
	}

	setting, err := s.GetSettings(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	oldValue, _ := s.serializeUserSetting(setting)

	var logEntries []domain.KvkkLog

	if req.Action == "REMIND_LATER" {
		setting.KvkkStatus = "PENDING"
		setting.PhotoConsent = "PENDING"
		setting.KvkkText = "PENDING"
		setting.PrivacyPolicy = "PENDING"
		setting.AntiBriberyPolicy = "PENDING"
		setting.KvkkApproved = false
		setting.KvkkApprovedAt = nil
		setting.KvkkRejectedAt = nil
		setting.KvkkLastPostponedAt = &now
		setting.ModifiedBy = "USER"

		logEntries = append(logEntries, domain.KvkkLog{
			UserID:       userID,
			DocumentType: "ALL",
			Action:       "REMIND_LATER",
			ClientIP:     clientIP,
			UserAgent:    userAgent,
		})
	} else { // SUBMIT
		// Apply changes
		setting.PhotoConsent = req.PhotoConsent
		setting.PhotoConsentAt = &now
		
		setting.KvkkText = req.KvkkText
		setting.KvkkTextAt = &now

		setting.PrivacyPolicy = req.PrivacyPolicy
		setting.PrivacyPolicyAt = &now

		setting.AntiBriberyPolicy = req.AntiBriberyPolicy
		setting.AntiBriberyPolicyAt = &now

		// Backward compatibility
		if req.PhotoConsent == "APPROVED" {
			setting.KvkkStatus = "APPROVED"
			setting.KvkkApproved = true
			setting.KvkkApprovedAt = &now
			setting.KvkkRejectedAt = nil
		} else {
			setting.KvkkStatus = "REJECTED"
			setting.KvkkApproved = false
			setting.KvkkApprovedAt = nil
			setting.KvkkRejectedAt = &now
		}
		setting.ModifiedBy = "USER"

		// Create separate log rows for each document
		logEntries = append(logEntries, domain.KvkkLog{
			UserID:       userID,
			DocumentType: "PHOTO_CONSENT",
			Action:       req.PhotoConsent,
			ClientIP:     clientIP,
			UserAgent:    userAgent,
		})
		logEntries = append(logEntries, domain.KvkkLog{
			UserID:       userID,
			DocumentType: "KVKK_TEXT",
			Action:       req.KvkkText,
			ClientIP:     clientIP,
			UserAgent:    userAgent,
		})
		logEntries = append(logEntries, domain.KvkkLog{
			UserID:       userID,
			DocumentType: "PRIVACY_POLICY",
			Action:       req.PrivacyPolicy,
			ClientIP:     clientIP,
			UserAgent:    userAgent,
		})
		logEntries = append(logEntries, domain.KvkkLog{
			UserID:       userID,
			DocumentType: "ANTI_BRIBERY_POLICY",
			Action:       req.AntiBriberyPolicy,
			ClientIP:     clientIP,
			UserAgent:    userAgent,
		})
	}

	// Update user settings in DB
	if err := s.repo.Update(setting); err != nil {
		return nil, err
	}

	newValue, _ := s.serializeUserSetting(setting)

	// Save all KvkkLogs
	for i := range logEntries {
		if err := s.repo.CreateKvkkLog(&logEntries[i]); err != nil {
			return nil, err
		}
	}

	// Create system audit log
	if s.auditService != nil {
		_ = s.auditService.CreateAuditLog(
			"UserSetting",
			setting.ID,
			"UPDATE_KVKK",
			oldValue,
			newValue,
			fmt.Sprintf("%d", userID),
		)
	}

	return setting, nil
}

// helper to serialize setting for audit logs
func (s *settingsService) serializeUserSetting(setting *domain.UserSetting) (string, error) {
	type settingSnap struct {
		PhotoConsent          string     `json:"photo_consent"`
		KvkkText              string     `json:"kvkk_text"`
		PrivacyPolicy         string     `json:"privacy_policy"`
		AntiBriberyPolicy     string     `json:"anti_bribery_policy"`
		PhotoConsentAt        *time.Time `json:"photo_consent_at"`
		KvkkTextAt            *time.Time `json:"kvkk_text_at"`
		PrivacyPolicyAt       *time.Time `json:"privacy_policy_at"`
		AntiBriberyPolicyAt   *time.Time `json:"anti_bribery_policy_at"`
		KvkkLastPostponedAt   *time.Time `json:"kvkk_last_postponed_at"`
		PromotionEmailAllowed bool       `json:"promotion_email_allowed"`
		PromotionSmsAllowed   bool       `json:"promotion_sms_allowed"`
	}
	snap := settingSnap{
		PhotoConsent:          setting.PhotoConsent,
		KvkkText:              setting.KvkkText,
		PrivacyPolicy:         setting.PrivacyPolicy,
		AntiBriberyPolicy:     setting.AntiBriberyPolicy,
		PhotoConsentAt:        setting.PhotoConsentAt,
		KvkkTextAt:            setting.KvkkTextAt,
		PrivacyPolicyAt:       setting.PrivacyPolicyAt,
		AntiBriberyPolicyAt:   setting.AntiBriberyPolicyAt,
		KvkkLastPostponedAt:   setting.KvkkLastPostponedAt,
		PromotionEmailAllowed: setting.PromotionEmailAllowed,
		PromotionSmsAllowed:   setting.PromotionSmsAllowed,
	}
	bytes, err := json.Marshal(snap)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
