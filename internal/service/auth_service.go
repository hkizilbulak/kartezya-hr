package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"kartezya-hr/internal/config"
	"kartezya-hr/internal/domain"
	"kartezya-hr/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

type AuthService interface {
	Login(email, password string) (*LoginResponse, error)
	ValidateToken(tokenString string) (*Claims, error)
	HashPassword(password string) (string, error)
	CheckPassword(hashedPassword, password string) error
	CreateUserForEmployee(email string, createdBy string) (*domain.User, error)
	GetYandexOAuthConfig() *oauth2.Config
	HandleYandexCallback(code string) (*LoginResponse, error)
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
	ID        uint     `json:"id"`
	Email     string   `json:"email"`
	FirstName string   `json:"firstName"`
	LastName  string   `json:"lastName"`
	Roles     []string `json:"roles"`
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

	// Get employee information to fetch firstName and lastName
	employee, err := s.userRepo.GetEmployeeByUserID(user.ID)
	firstName := ""
	lastName := ""
	if err == nil && employee != nil {
		firstName = employee.FirstName
		lastName = employee.LastName
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
			ID:        user.ID,
			Email:     user.Email,
			FirstName: firstName,
			LastName:  lastName,
			Roles:     roleNames,
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

// YandexUserInfo represents the user information returned by Yandex OAuth API
type YandexUserInfo struct {
	ID            string `json:"id"`
	Login         string `json:"login"`
	DefaultEmail  string `json:"default_email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	DisplayName   string `json:"display_name"`
	RealName      string `json:"real_name"`
	DefaultAvatar string `json:"default_avatar_id"`
}

// GetYandexOAuthConfig returns the OAuth2 configuration for Yandex
func (s *authService) GetYandexOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.config.OAuth.YandexClientID,
		ClientSecret: s.config.OAuth.YandexClientSecret,
		RedirectURL:  s.config.OAuth.YandexRedirectURL,
		Scopes:       []string{"login:email", "login:info"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://oauth.yandex.com/authorize",
			TokenURL: "https://oauth.yandex.com/token",
		},
	}
}

// HandleYandexCallback handles the OAuth callback from Yandex
func (s *authService) HandleYandexCallback(code string) (*LoginResponse, error) {
	ctx := context.Background()
	oauthConfig := s.GetYandexOAuthConfig()

	// Exchange the authorization code for an access token
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}

	// Get user information from Yandex
	client := oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://login.yandex.ru/info?format=json")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: status %d, body: %s", resp.StatusCode, string(body))
	}

	var yandexUser YandexUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&yandexUser); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Check if email is available
	if yandexUser.DefaultEmail == "" {
		return nil, errors.New("email not provided by Yandex")
	}

	// Check if user exists, if not create a new user
	user, err := s.userRepo.GetByEmail(yandexUser.DefaultEmail)
	if err != nil {
		// User doesn't exist, create a new one
		// Generate a random password (user won't need it for OAuth login)
		randomPassword, err := s.HashPassword("oauth-" + yandexUser.ID + "-" + time.Now().String())
		if err != nil {
			return nil, err
		}

		newUser := &domain.User{
			Email:    yandexUser.DefaultEmail,
			Password: randomPassword,
		}

		if err := s.userRepo.Create(newUser, "yandex-oauth"); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		user = newUser

		// Audit the creation
		userForAudit := *newUser
		userForAudit.Password = "[REDACTED]"
		if err := s.auditService.CreateAuditLog("User", newUser.ID, domain.AuditActionCreate, nil, &userForAudit, "yandex-oauth"); err != nil {
			// Log error but don't fail the operation
		}
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

	// Get employee information to fetch firstName and lastName
	employee, err := s.userRepo.GetEmployeeByUserID(user.ID)
	firstName := yandexUser.FirstName
	lastName := yandexUser.LastName
	if err == nil && employee != nil {
		// Prefer employee data if available
		if employee.FirstName != "" {
			firstName = employee.FirstName
		}
		if employee.LastName != "" {
			lastName = employee.LastName
		}
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

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := jwtToken.SignedString([]byte(s.config.JWT.Secret))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		User: &UserWithRoles{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: firstName,
			LastName:  lastName,
			Roles:     roleNames,
		},
	}, nil
}
