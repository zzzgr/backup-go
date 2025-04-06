package config

import (
	"backup-go/entity"
	"fmt"
	"os"
	"sync"

	"github.com/glebarez/sqlite"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 全局配置和数据库连接
var (
	appConfig *Configuration
	db        *gorm.DB
	configMux sync.RWMutex
)

// Configuration 配置结构
type Configuration struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		Type     string `yaml:"type"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Db       string `yaml:"db"`
	} `yaml:"database"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	configMux.Lock()
	defer configMux.Unlock()

	// 设置默认配置
	setDefaultConfig()

	// 尝试从文件加载配置
	if _, err := os.Stat(configPath); err == nil {
		file, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}

		err = yaml.Unmarshal(file, &appConfig)
		if err != nil {
			return fmt.Errorf("解析配置文件失败: %w", err)
		}
	}

	return nil
}

// 设置默认配置
func setDefaultConfig() {
	appConfig = &Configuration{}
	appConfig.Server.Host = "localhost"
	appConfig.Server.Port = 8080

	appConfig.Database.Type = "sqlite"
	appConfig.Database.Host = "localhost"
	appConfig.Database.Port = 3306
	appConfig.Database.User = "root"
	appConfig.Database.Password = "password"
	appConfig.Database.Db = "backup_go"

}

// Get 获取配置
func Get() *Configuration {
	configMux.RLock()
	defer configMux.RUnlock()
	return appConfig
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return db
}

// InitDB 初始化数据库连接
func InitDB() error {
	var err error

	// 创建GORM的静默日志配置
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	switch appConfig.Database.Type {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			appConfig.Database.User,
			appConfig.Database.Password,
			appConfig.Database.Host,
			appConfig.Database.Port,
			appConfig.Database.Db,
		)
		fmt.Printf("连接MySQL数据库: %s\n", dsn)
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	case "sqlite":

		fmt.Printf("连接SQLite数据库: %s\n", "backup.db")
		db, err = gorm.Open(sqlite.Open("backup.db"), gormConfig)
	default:
		return fmt.Errorf("不支持的数据库类型: %s", appConfig.Database.Type)
	}

	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	return nil
}

// MigrateDB 执行数据库迁移
func MigrateDB() error {
	// 自动迁移表结构
	err := db.AutoMigrate(
		&entity.BackupTask{},
		&entity.BackupRecord{},
		&entity.SystemConfig{},
	)
	if err != nil {
		return fmt.Errorf("数据表迁移失败: %w", err)
	}
	return nil
}
