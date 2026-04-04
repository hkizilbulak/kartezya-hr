package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// LocalStorageProvider implements StorageProvider for local filesystem
type LocalStorageProvider struct {
	basePath string
	baseURL  string
}

// NewLocalStorageProvider creates a new local storage provider
func NewLocalStorageProvider(basePath string, baseURL string) *LocalStorageProvider {
	return &LocalStorageProvider{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// Upload saves file to local filesystem
func (p *LocalStorageProvider) Upload(file multipart.File, path string) error {
	fullPath := filepath.Join(p.basePath, path)

	// Create directory if not exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

// Download reads file from local filesystem
func (p *LocalStorageProvider) Download(path string) ([]byte, error) {
	fullPath := filepath.Join(p.basePath, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}

// Delete removes file from local filesystem
func (p *LocalStorageProvider) Delete(path string) error {
	fullPath := filepath.Join(p.basePath, path)
	if err := os.Remove(fullPath); err != nil {
		// Ignore not exists errors
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}
	return nil
}

// GeneratePresignedURL generates a temporary URL for file access
// For local storage, this returns a direct URL (in production, use cloud storage with real pre-signed URLs)
func (p *LocalStorageProvider) GeneratePresignedURL(path string, expiryMinutes int) (string, error) {
	// In production with cloud storage (S3/Azure), this would generate a real pre-signed URL
	// For local development, we just return the direct URL
	// You should implement a token-based URL in production for security

	url := fmt.Sprintf("%s/uploads/%s?expires=%d",
		p.baseURL,
		path,
		time.Now().Add(time.Duration(expiryMinutes)*time.Minute).Unix(),
	)

	return url, nil
}
