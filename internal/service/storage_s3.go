package service

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3StorageProvider implements StorageProvider for S3-compatible storage
// Works with AWS S3, Backblaze B2, MinIO, DigitalOcean Spaces, etc.
type S3StorageProvider struct {
	client   *s3.S3
	bucket   string
	basePath string // Base path prefix in bucket (e.g., "documents", "attachments/hr")
}

// NewS3StorageProvider creates a new S3-compatible storage provider
// endpoint: S3-compatible endpoint URL (e.g., "https://s3.eu-central-003.backblazeb2.com")
//
//	Leave empty for AWS S3 (uses default AWS endpoints)
//
// region: AWS region or compatible region (e.g., "eu-central-003" for Backblaze)
// bucket: Bucket name
// basePath: Base path prefix in bucket (e.g., "documents", "attachments/hr")
// accessKey: Access Key ID
// secretKey: Secret Access Key
func NewS3StorageProvider(endpoint, region, bucket, basePath, accessKey, secretKey string) (*S3StorageProvider, error) {
	// Configure AWS session
	config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Region:      aws.String(region),
	}

	// Set custom endpoint if provided (for Backblaze B2, MinIO, etc.)
	if endpoint != "" {
		config.Endpoint = aws.String(endpoint)
		config.S3ForcePathStyle = aws.Bool(true) // Required for some S3-compatible services
	}

	// Create session
	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 session: %w", err)
	}

	return &S3StorageProvider{
		client:   s3.New(sess),
		bucket:   bucket,
		basePath: basePath,
	}, nil
}

// Upload saves file to S3-compatible storage
func (p *S3StorageProvider) Upload(file multipart.File, path string) error {
	// Read file content into buffer
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// Combine base path with file path
	fullPath := path
	if p.basePath != "" {
		fullPath = fmt.Sprintf("%s/%s", p.basePath, path)
	}

	// Upload to S3
	_, err := p.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(fullPath),
		Body:   bytes.NewReader(buf.Bytes()),
		ACL:    aws.String("private"), // Private by default, use pre-signed URLs for access
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

// Download reads file from S3-compatible storage
func (p *S3StorageProvider) Download(path string) ([]byte, error) {
	// Combine base path with file path
	fullPath := path
	if p.basePath != "" {
		fullPath = fmt.Sprintf("%s/%s", p.basePath, path)
	}

	result, err := p.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(fullPath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object: %w", err)
	}

	return data, nil
}

// Delete removes file from S3-compatible storage
func (p *S3StorageProvider) Delete(path string) error {
	// Combine base path with file path
	fullPath := path
	if p.basePath != "" {
		fullPath = fmt.Sprintf("%s/%s", p.basePath, path)
	}

	_, err := p.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(fullPath),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}

	return nil
}

// GeneratePresignedURL generates a pre-signed URL for temporary file access
func (p *S3StorageProvider) GeneratePresignedURL(path string, expiryMinutes int) (string, error) {
	// Combine base path with file path
	fullPath := path
	if p.basePath != "" {
		fullPath = fmt.Sprintf("%s/%s", p.basePath, path)
	}

	req, _ := p.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(fullPath),
	})

	// Generate pre-signed URL with expiration
	url, err := req.Presign(time.Duration(expiryMinutes) * time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to generate pre-signed URL: %w", err)
	}

	return url, nil
}
