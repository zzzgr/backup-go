package repository

import (
	"backup-go/config"
	"backup-go/entity"
	"errors"
	"gorm.io/gorm"
)

// ConfigRepository 配置仓库
type ConfigRepository struct {
	db *gorm.DB
}

// NewConfigRepository 创建仓库实例
func NewConfigRepository() *ConfigRepository {
	return &ConfigRepository{
		db: config.GetDB(),
	}
}

// FindAll 查询所有配置
func (r *ConfigRepository) FindAll() ([]*entity.SystemConfig, error) {
	var configs []*entity.SystemConfig
	if err := r.db.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// FindByID 根据ID查询配置
func (r *ConfigRepository) FindByID(id uint) (*entity.SystemConfig, error) {
	var config entity.SystemConfig
	if err := r.db.Where("id = ?", id).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("配置不存在")
		}
		return nil, err
	}
	return &config, nil
}

// FindByKey 根据键查询配置
func (r *ConfigRepository) FindByKey(key string) (*entity.SystemConfig, error) {
	var config entity.SystemConfig
	if err := r.db.Where("config_key = ?", key).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("配置不存在")
		}
		return nil, err
	}
	return &config, nil
}

// Create 创建配置
func (r *ConfigRepository) Create(config *entity.SystemConfig) error {
	return r.db.Create(config).Error
}

// Update 更新配置
func (r *ConfigRepository) Update(config *entity.SystemConfig) error {
	return r.db.Save(config).Error
}

// Delete 删除配置
func (r *ConfigRepository) Delete(id uint) error {
	return r.db.Delete(&entity.SystemConfig{}, id).Error
}
