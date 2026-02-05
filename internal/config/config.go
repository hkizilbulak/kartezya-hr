package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	JWT      JWTConfig
	Server   ServerConfig
	App      AppConfig
	Email    EmailConfig
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
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string
	FromName     string
	FrontendURL  string
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
			SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort:     smtpPort,
			SMTPUser:     getEnv("SMTP_USER", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "info@kartezya.com"),
			FromName:     getEnv("FROM_NAME", "Kartezya Teknoloji"),
			FrontendURL:  getEnv("FRONTEND_URL", "http://localhost:3000"),
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
