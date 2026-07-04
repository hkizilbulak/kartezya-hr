package service

import (
	"fmt"
	"strings"

	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"
)

type MailConfigService interface {
	GetAll() ([]domain.MailConfiguration, error)
	GetByID(id uint) (*domain.MailConfiguration, error)
	Create(cfg *domain.MailConfiguration) error
	Update(id uint, cfg *domain.MailConfiguration) error
	Delete(id uint) error
	// ResolveRecipients returns (to, cc, bcc, templateCode) for a mail key.
	// Only STATIC recipients are returned; DYNAMIC placeholders need caller-side context.
	ResolveRecipients(mailKey string) (to []string, cc []string, bcc []string, templateCode string, err error)
}

type mailConfigService struct {
	repo repository.MailConfigRepository
}

func NewMailConfigService(repo repository.MailConfigRepository) MailConfigService {
	return &mailConfigService{repo: repo}
}

func (s *mailConfigService) GetAll() ([]domain.MailConfiguration, error) {
	return s.repo.GetAll()
}

func (s *mailConfigService) GetByID(id uint) (*domain.MailConfiguration, error) {
	return s.repo.GetByID(id)
}

func (s *mailConfigService) Create(cfg *domain.MailConfiguration) error {
	if cfg.Provider != domain.MailProviderResend && cfg.Provider != domain.MailProviderSMTP {
		return fmt.Errorf("invalid provider: must be RESEND or SMTP")
	}
	if cfg.Provider == domain.MailProviderResend && cfg.ResendTemplateCode == "" {
		return fmt.Errorf("resend_template_code is required when provider is RESEND")
	}
	return s.repo.Create(cfg)
}

func (s *mailConfigService) Update(id uint, cfg *domain.MailConfiguration) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("mail configuration not found")
	}
	if cfg.Provider != domain.MailProviderResend && cfg.Provider != domain.MailProviderSMTP {
		return fmt.Errorf("invalid provider: must be RESEND or SMTP")
	}
	if cfg.Provider == domain.MailProviderResend && cfg.ResendTemplateCode == "" {
		return fmt.Errorf("resend_template_code is required when provider is RESEND")
	}

	existing.Description = cfg.Description
	existing.Provider = cfg.Provider
	existing.ResendTemplateCode = cfg.ResendTemplateCode
	existing.IsActive = cfg.IsActive

	if err := s.repo.Update(existing); err != nil {
		return err
	}
	return s.repo.ReplaceRecipients(id, cfg.Recipients)
}

func (s *mailConfigService) Delete(id uint) error {
	if _, err := s.repo.GetByID(id); err != nil {
		return fmt.Errorf("mail configuration not found")
	}
	return s.repo.Delete(id)
}

// ResolveRecipients resolves only STATIC addresses from the DB config.
// DYNAMIC placeholders (e.g. {{TRIGGER_USER}}) are skipped — callers resolve them if needed.
func (s *mailConfigService) ResolveRecipients(mailKey string) (to []string, cc []string, bcc []string, templateCode string, err error) {
	cfg, err := s.repo.GetByKey(mailKey)
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("mail config not found for key %q: %w", mailKey, err)
	}
	if !cfg.IsActive {
		return nil, nil, nil, "", fmt.Errorf("mail config %q is inactive", mailKey)
	}
	templateCode = cfg.ResendTemplateCode
	for _, r := range cfg.Recipients {
		if r.ValueType != domain.ValueTypeStatic {
			continue
		}
		addrs := strings.FieldsFunc(r.RecipientValue, func(c rune) bool {
			return c == ',' || c == ';'
		})
		switch r.RecipientType {
		case domain.RecipientTypeTo:
			for _, addr := range addrs {
				if trimmed := strings.TrimSpace(addr); trimmed != "" {
					to = append(to, trimmed)
				}
			}
		case domain.RecipientTypeCC:
			for _, addr := range addrs {
				if trimmed := strings.TrimSpace(addr); trimmed != "" {
					cc = append(cc, trimmed)
				}
			}
		case domain.RecipientTypeBCC:
			for _, addr := range addrs {
				if trimmed := strings.TrimSpace(addr); trimmed != "" {
					bcc = append(bcc, trimmed)
				}
			}
		}
	}
	return to, cc, bcc, templateCode, nil
}
