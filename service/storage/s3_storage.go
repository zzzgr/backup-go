package storage

import (
	"backup-go/entity"
	configService "backup-go/service/config"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3StorageService S3协议存储服务
type S3StorageService struct {
	session    *session.Session
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	s3Client   *s3.S3
	bucketName string
}

// NewS3StorageService 创建S3存储服务
func NewS3StorageService() *S3StorageService {
	// 从系统配置表获取配置
	cs := configService.NewConfigService()

	// 读取S3配置
	s3Endpoint, _ := cs.GetConfigValue("storage.s3Endpoint")
	s3Region, _ := cs.GetConfigValue("storage.s3Region")
	s3AccessKey, _ := cs.GetConfigValue("storage.s3AccessKey")
	s3SecretKey, _ := cs.GetConfigValue("storage.s3SecretKey")
	s3Bucket, _ := cs.GetConfigValue("storage.s3Bucket")

	// 如果配置为空，则使用默认值
	if s3Endpoint == "" {
		s3Endpoint = "" // 默认为空
	}
	if s3Region == "" {
		s3Region = "us-east-1" // 默认区域
	}
	if s3AccessKey == "" {
		s3AccessKey = "" // 默认为空
	}
	if s3SecretKey == "" {
		s3SecretKey = "" // 默认为空
	}
	if s3Bucket == "" {
		s3Bucket = "backup-go" // 默认存储桶
	}

	service := &S3StorageService{
		bucketName: s3Bucket,
	}

	// 创建AWS会话
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(s3Region),
		Endpoint:         aws.String(s3Endpoint),
		Credentials:      credentials.NewStaticCredentials(s3AccessKey, s3SecretKey, ""),
		DisableSSL:       aws.Bool(false),
		S3ForcePathStyle: aws.Bool(true), // 支持非AWS S3服务
	})
	if err != nil {
		fmt.Printf("Failed to create S3 session: %v\n", err)
		return service
	}
	service.session = sess

	// 创建上传器、下载器和客户端
	service.uploader = s3manager.NewUploader(sess)
	service.downloader = s3manager.NewDownloader(sess)
	service.s3Client = s3.New(sess)

	return service
}

// Save 保存文件
func (s *S3StorageService) Save(filename string, content io.Reader) (string, error) {
	if s.session == nil {
		return "", fmt.Errorf("S3 not configured properly")
	}

	// 创建目录格式
	today := time.Now().Format("20060102")
	s3Path := filepath.Join("backups", today, filename)

	// 上传文件
	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Path),
		Body:   content,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return s3Path, nil
}

// Get 获取文件
func (s *S3StorageService) Get(path string) (io.ReadCloser, error) {
	if s.session == nil {
		return nil, fmt.Errorf("S3 not configured properly")
	}

	// 直接使用GetObject方法获取对象
	result, err := s.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file from S3: %w", err)
	}

	return result.Body, nil
}

// Delete 删除文件
func (s *S3StorageService) Delete(path string) error {
	if s.session == nil {
		return fmt.Errorf("S3 not configured properly")
	}

	// 删除文件
	_, err := s.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// GetStorageType 获取存储类型
func (s *S3StorageService) GetStorageType() entity.StorageType {
	return entity.S3Storage
}
