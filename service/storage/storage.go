package storage

import (
	"backup-go/entity"
	configService "backup-go/service/config"
	"io"
)

// StorageService 存储服务接口
type StorageService interface {
	// Save 保存文件
	Save(filename string, content io.Reader) (string, error)

	// Get 获取文件
	Get(path string) (io.ReadCloser, error)

	// Delete 删除文件
	Delete(path string) error

	// GetStorageType 获取存储类型
	GetStorageType() entity.StorageType
}

// 存储服务工厂
func NewStorageService(storageType entity.StorageType) (StorageService, error) {
	// 如果未指定存储类型，从系统配置表读取
	if storageType == "" {
		cs := configService.NewConfigService()
		storageTypeStr, err := cs.GetConfigValue("storage.type")
		if err != nil || storageTypeStr == "" {
			// 默认使用本地存储
			storageType = entity.LocalStorage
		} else {
			storageType = entity.StorageType(storageTypeStr)
		}
	}

	// 根据存储类型创建相应的存储服务
	switch storageType {
	case entity.LocalStorage:
		return NewLocalStorageService(), nil
	case entity.S3Storage:
		return NewS3StorageService(), nil
	default:
		return NewLocalStorageService(), nil
	}
}
