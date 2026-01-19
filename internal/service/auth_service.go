package service

import (
	"errors"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Login(email, password string) (*LoginResponse, error)
	ValidateToken(tokenString string) (*Claims, error)
	HashPassword(password string) (string, error)
	CheckPassword(hashedPassword, password string) error
	CreateUserForEmployee(email string, createdBy string) (*domain.User, error)
}

type authService struct {
	userRepo     repository.UserRepository
	userRoleRepo repository.UserRoleRepository
	roleRepo     repository.RoleRepository
	auditService AuditService
	config       *config.Config
}

type LoginResponse struct {
	Token     string         `json:"token"`
	ExpiresAt time.Time      `json:"expires_at"`
	User      *UserWithRoles `json:"user"`
}

type UserWithRoles struct {
	*domain.User
	Roles []string `json:"roles"`
}

type Claims struct {
	UserID uint     `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func NewAuthService(userRepo repository.UserRepository, userRoleRepo repository.UserRoleRepository, roleRepo repository.RoleRepository, auditService AuditService, config *config.Config) AuthService {
	return &authService{
		userRepo:     userRepo,
		userRoleRepo: userRoleRepo,
		roleRepo:     roleRepo,
		auditService: auditService,
		config:       config,
	}
}

func (s *authService) Login(email, password string) (*LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := s.CheckPassword(user.Password, password); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Get user roles
	roles, err := s.userRoleRepo.GetRolesByUserID(user.ID)
	if err != nil {
		return nil, err
	}

	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	// Generate JWT token
	expiresAt := time.Now().Add(s.config.JWT.ExpiryHours)
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Roles:  roleNames,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.config.App.Name,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: &UserWithRoles{
			User:  user,
			Roles: roleNames,
		},
	}, nil
}

func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s *authService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *authService) CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (s *authService) CreateUser(user *domain.User, createdBy string) error {
	// Hash password if provided
	if user.Password != "" {
		hashedPassword, err := s.HashPassword(user.Password)
		if err != nil {
			return err
		}
		user.Password = hashedPassword
	}

	// Create the user
	if err := s.userRepo.Create(user, createdBy); err != nil {
		return err
	}

	// Create a copy for audit without the password
	userForAudit := *user
	userForAudit.Password = "[REDACTED]" // Don't log passwords in audit

	// Audit the creation
	if err := s.auditService.CreateAuditLog("User", user.ID, domain.AuditActionCreate, nil, &userForAudit, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return nil
}

// CreateUserForEmployee creates a user specifically for employee registration
// Uses default password "employee123" for new employee accounts
func (s *authService) CreateUserForEmployee(email string, createdBy string) (*domain.User, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(email)
	if err == nil {
		// User exists, return existing user
		return existingUser, nil
	}

	// Create new user with default password
	defaultPassword := "employee123"
	hashedPassword, err := s.HashPassword(defaultPassword)
	if err != nil {
		return nil, err
	}

	newUser := &domain.User{
		Email:    email,
		Password: hashedPassword,
	}

	// Create the user
	if err := s.userRepo.Create(newUser, createdBy); err != nil {
		return nil, err
	}

	// Create a copy for audit without the password
	userForAudit := *newUser
	userForAudit.Password = "[REDACTED]"

	// Audit the creation
	if err := s.auditService.CreateAuditLog("User", newUser.ID, domain.AuditActionCreate, nil, &userForAudit, createdBy); err != nil {
		// Log error but don't fail the operation
	}

	return newUser, nil
}
