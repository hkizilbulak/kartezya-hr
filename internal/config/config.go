package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database    DatabaseConfig
	JWT         JWTConfig
	Server      ServerConfig
	App         AppConfig
	Email       EmailConfig
	OAuth       OAuthConfig
	Storage     StorageConfig
	ReportEmail ReportEmailConfig
}

type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	Name        string
	SSLMode     string
	Debug       bool
	TablePrefix string
}

type JWTConfig struct {
	Secret      string
	ExpiryHours time.Duration
}

type ServerConfig struct {
	Port    string
	GinMode string
}

type AppConfig struct {
	Name    string
	Version string
}

type EmailConfig struct {
	SMTPHost             string
	SMTPPort             int
	SMTPUser             string
	SMTPPassword         string
	FromEmail            string
	FromName             string
	FrontendURL          string
	Provider             string // "smtp" or "resend"
	ResendAPIKey         string
	EventAllCompanyGroup []string // Mail group addresses for all-company events (comma-separated)
}

type OAuthConfig struct {
	YandexClientID     string
	YandexClientSecret string
	YandexRedirectURL  string
}

type StorageConfig struct {
	Provider string // "local", "s3", "backblaze", "azure"
	BasePath string // Local storage base path
	BaseURL  string // Base URL for file access

	// S3/Backblaze B2 Configuration
	S3Endpoint  string // S3-compatible endpoint (e.g., Backblaze B2)
	S3Bucket    string // Bucket name
	S3Region    string // Region (optional for some providers)
	S3BasePath  string // Base path prefix in bucket (e.g., "documents", "attachments/hr")
	S3AccessKey string // Access key ID
	S3SecretKey string // Secret access key

	// Azure Blob Configuration (future use)
	AzureAccount   string
	AzureContainer string
	AzureAccessKey string
}

type ReportEmailConfig struct {
	WorkDayRecipients  []string
	EffortRecipients   []string
	ContractRecipients []string
	GradeRecipients    []string
}

func Load() *Config {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	port, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		log.Fatal("Invalid DB_PORT")
	}

	expiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		log.Fatal("Invalid JWT_EXPIRY_HOURS")
	}

	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		log.Fatal("Invalid SMTP_PORT")
	}

	return &Config{
		Database: DatabaseConfig{
			Host:        getEnv("DB_HOST", "localhost"),
			Port:        port,
			User:        getEnv("DB_USER", "postgres"),
			Password:    getEnv("DB_PASSWORD", "postgres"),
			Name:        getEnv("DB_NAME", "kartezya_hr"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
			Debug:       getEnv("DB_DEBUG", "false") == "true",
			TablePrefix: getEnv("DB_TABLE_PREFIX", "hr"),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "default-secret-change-me"),
			ExpiryHours: time.Duration(expiryHours) * time.Hour,
		},
		Server: ServerConfig{
			Port:    getEnv("SERVER_PORT", "8080"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		App: AppConfig{
			Name:    getEnv("APP_NAME", "Kartezya HR"),
			Version: getEnv("APP_VERSION", "1.0.0"),
		},
		Email: EmailConfig{
			SMTPHost:             getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:             smtpPort,
			SMTPUser:             getEnv("SMTP_USER", ""),
			SMTPPassword:         getEnv("SMTP_PASSWORD", ""),
			FromEmail:            getEnv("FROM_EMAIL", "info@kartezya.com"),
			FromName:             getEnv("FROM_NAME", "Kartezya Teknoloji"),
			FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:3000"),
			Provider:             getEnv("EMAIL_PROVIDER", "smtp"), // "smtp" or "resend"
			ResendAPIKey:         getEnv("RESEND_API_KEY", ""),
			EventAllCompanyGroup: parseEmailList(getEnv("EVENT_EMAIL_ALL_COMPANY", "")),
		},
		OAuth: OAuthConfig{
			YandexClientID:     getEnv("YANDEX_CLIENT_ID", ""),
			YandexClientSecret: getEnv("YANDEX_CLIENT_SECRET", ""),
			YandexRedirectURL:  getEnv("YANDEX_REDIRECT_URL", "http://localhost:8080/api/v1/auth/yandex/callback"),
		},
		ReportEmail: ReportEmailConfig{
			WorkDayRecipients:  parseEmailList(getEnv("REPORT_EMAIL_WORK_DAY", "huseyinkizilbulak76@gmail.com")),
			EffortRecipients:   parseEmailList(getEnv("REPORT_EMAIL_EFFORT", "huseyinkizilbulak76@gmail.com")),
			ContractRecipients: parseEmailList(getEnv("REPORT_EMAIL_CONTRACT", "huseyinkizilbulak76@gmail.com")),
			GradeRecipients:    parseEmailList(getEnv("REPORT_EMAIL_GRADE", "huseyinkizilbulak76@gmail.com")),
		},
		Storage: StorageConfig{
			Provider:    getEnv("STORAGE_PROVIDER", "local"), // Options: local, s3, backblaze, azure
			BasePath:    getEnv("STORAGE_BASE_PATH", "./uploads"),
			BaseURL:     getEnv("STORAGE_BASE_URL", "http://localhost:8080"),
			S3Endpoint:  getEnv("S3_ENDPOINT", ""), // e.g., https://s3.eu-central-003.backblazeb2.com
			S3Bucket:    getEnv("S3_BUCKET", ""),
			S3Region:    getEnv("S3_REGION", ""),
			S3BasePath:  getEnv("S3_BASE_PATH", "documents"), // Base path prefix in bucket
			S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
			S3SecretKey: getEnv("S3_SECRET_KEY", ""),
		},
	}
}

func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetTableName returns the table name with prefix if configured
func (c *Config) GetTableName(tableName string) string {
	if c.Database.TablePrefix != "" {
		// If table already has hr_ prefix, replace it with the configured prefix
		if len(tableName) > 3 && tableName[:3] == "hr_" {
			return c.Database.TablePrefix + "_" + tableName[3:]
		}
		// If no hr_ prefix, add the configured prefix
		return c.Database.TablePrefix + "_" + tableName
	}
	// If no prefix configured, return original table name
	return tableName
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseEmailList(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := []string{}
	for _, item := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// GetRecipients returns recipients for a given report type
func (c *ReportEmailConfig) GetRecipients(reportType string) []string {
	switch reportType {
	case "work-day":
		return c.WorkDayRecipients
	case "effort":
		return c.EffortRecipients
	case "contract":
		return c.ContractRecipients
	case "grade":
		return c.GradeRecipients
	default:
		return []string{}
	}
}
