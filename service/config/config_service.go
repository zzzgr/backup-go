package config

import (
	"backup-go/entity"
	"backup-go/repository"
)

// ConfigService 配置服务
type ConfigService struct {
	repo *repository.ConfigRepository
}

// NewConfigService 创建配置服务
func NewConfigService() *ConfigService {
	return &ConfigService{
		repo: repository.NewConfigRepository(),
	}
}

// InitDefaultConfigs 初始化默认配置
func (s *ConfigService) InitDefaultConfigs() error {
	// 要初始化的默认配置
	defaultConfigs := []struct {
		Key         string
		Value       string
		Description string
	}{
		{"system.password", "123456", "系统登录密码"},
		{"storage.type", "local", "存储类型"},
		{"storage.localPath", "./backup", "本地存储路径"},
		// 添加S3相关配置
		{"storage.s3Endpoint", "", "S3服务端点"},
		{"storage.s3Region", "us-east-1", "S3区域"},
		{"storage.s3AccessKey", "", "S3访问密钥"},
		{"storage.s3SecretKey", "", "S3私有密钥"},
		{"storage.s3Bucket", "", "S3存储桶名称"},
		// 添加系统自动清理配置
		{"system.autoCleanupDays", "90", "自动清理天数，0表示不清理"},
		// 添加Webhook相关配置
		{"webhook.enabled", "false", "是否启用Webhook通知"},
		{"webhook.url", "", "Webhook URL"},
		{"webhook.headers", "", "Webhook请求头，一行一个"},
		{"webhook.body", `{"event":"${event}","taskName":"${taskName}","message":"${message}"}`, "Webhook请求体模板"},
		// 添加站点配置
		{"system.siteName", "备份系统", "站点名称"},
	}

	// 逐个检查配置是否存在，不存在则创建
	for _, cfg := range defaultConfigs {
		// 检查配置是否存在
		_, err := s.GetConfigByKey(cfg.Key)
		if err != nil {
			// 配置不存在，创建默认配置
			config := &entity.SystemConfig{
				ConfigKey:   cfg.Key,
				ConfigValue: cfg.Value,
				Description: cfg.Description,
			}
			if err := s.CreateConfig(config); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetAllConfigs 获取所有配置
func (s *ConfigService) GetAllConfigs() ([]*entity.SystemConfig, error) {
	return s.repo.FindAll()
}

// GetConfig 获取配置
func (s *ConfigService) GetConfig(id uint) (*entity.SystemConfig, error) {
	return s.repo.FindByID(id)
}

// GetConfigByKey 通过key获取配置
func (s *ConfigService) GetConfigByKey(key string) (*entity.SystemConfig, error) {
	return s.repo.FindByKey(key)
}

// CreateConfig 创建配置
func (s *ConfigService) CreateConfig(config *entity.SystemConfig) error {
	return s.repo.Create(config)
}

// UpdateConfig 更新配置
func (s *ConfigService) UpdateConfig(config *entity.SystemConfig) error {
	return s.repo.Update(config)
}

// DeleteConfig 删除配置
func (s *ConfigService) DeleteConfig(id uint) error {
	return s.repo.Delete(id)
}

// GetConfigValue 获取配置值
func (s *ConfigService) GetConfigValue(key string) (string, error) {
	config, err := s.GetConfigByKey(key)
	if err != nil {
		return "", err
	}
	return config.ConfigValue, nil
}

// SetConfigValue 设置配置值
func (s *ConfigService) SetConfigValue(key string, value string, description string) error {
	// 先检查是否存在
	config, err := s.GetConfigByKey(key)
	if err != nil {
		// 不存在则创建
		if err.Error() == "配置不存在" {
			newConfig := &entity.SystemConfig{
				ConfigKey:   key,
				ConfigValue: value,
				Description: description,
			}
			return s.CreateConfig(newConfig)
		}
		return err
	}

	// 存在则更新
	config.ConfigValue = value
	if description != "" {
		config.Description = description
	}
	return s.UpdateConfig(config)
}

// GetOrCreateConfig 获取或创建配置
func (s *ConfigService) GetOrCreateConfig(key string, defaultValue string, description string) (*entity.SystemConfig, error) {
	config, err := s.GetConfigByKey(key)
	if err != nil {
		// 配置不存在，创建新配置
		config = &entity.SystemConfig{
			ConfigKey:   key,
			ConfigValue: defaultValue,
			Description: description,
		}
		if err := s.CreateConfig(config); err != nil {
			return nil, err
		}
		return config, nil
	}
	return config, nil
}

// GetConfigValueByKey 获取配置值，如果不存在则返回错误
func (s *ConfigService) GetConfigValueByKey(key string) (string, error) {
	config, err := s.GetConfigByKey(key)
	if err != nil {
		return "", err
	}
	return config.ConfigValue, nil
}

// GetConfigValueOrDefault 获取配置值，如果不存在则返回默认值
func (s *ConfigService) GetConfigValueOrDefault(key string, defaultValue string) string {
	value, err := s.GetConfigValueByKey(key)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}
