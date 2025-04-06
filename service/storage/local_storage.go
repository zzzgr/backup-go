package storage

import (
	"backup-go/entity"
	configService "backup-go/service/config"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LocalStorageService 本地存储服务
type LocalStorageService struct {
	basePath string
}

// NewLocalStorageService 创建本地存储服务
func NewLocalStorageService() *LocalStorageService {
	// 从系统配置获取本地存储路径
	cs := configService.NewConfigService()
	localPath, err := cs.GetConfigValue("storage.localPath")

	// 如果获取失败，则使用默认配置
	if err != nil || localPath == "" {
		localPath = "./backups"
	}

	// 确保目录存在
	if err := os.MkdirAll(localPath, 0755); err != nil {
		// 使用默认路径
		if err := os.MkdirAll("./backups", 0755); err != nil {
			fmt.Printf("Failed to create backup directory: %v\n", err)
		}
		return &LocalStorageService{basePath: "./backups"}
	}
	return &LocalStorageService{basePath: localPath}
}

// Save 保存文件
func (s *LocalStorageService) Save(filename string, content io.Reader) (string, error) {
	// 创建目录
	today := time.Now().Format("20060102")
	dirPath := filepath.Join(s.basePath, today)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建文件路径
	filePath := filepath.Join(dirPath, filename)

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 写入文件
	_, err = io.Copy(file, content)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// 返回相对路径
	relativePath, err := filepath.Rel(s.basePath, filePath)
	if err != nil {
		return filePath, nil
	}
	return relativePath, nil
}

// Get 获取文件
func (s *LocalStorageService) Get(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}

// Delete 删除文件
func (s *LocalStorageService) Delete(path string) error {
	fullPath := filepath.Join(s.basePath, path)
	err := os.Remove(fullPath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetStorageType 获取存储类型
func (s *LocalStorageService) GetStorageType() entity.StorageType {
	return entity.LocalStorage
}
