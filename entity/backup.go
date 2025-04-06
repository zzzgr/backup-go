package entity

import (
	"time"
)

// BackupType 备份类型
type BackupType string

const (
	DatabaseBackup BackupType = "database" // 数据库备份
	FileBackup     BackupType = "file"     // 文件备份
	ConfigBackup   BackupType = "config"   // 配置文件备份
)

// BackupStatus 备份状态
type BackupStatus string

const (
	StatusPending   BackupStatus = "pending"   // 等待中
	StatusRunning   BackupStatus = "running"   // 执行中
	StatusSuccess   BackupStatus = "success"   // 成功
	StatusFailed    BackupStatus = "failed"    // 失败
	StatusCancelled BackupStatus = "cancelled" // 已取消
	StatusCleaned   BackupStatus = "cleaned"   // 已清理
)

// StorageType 存储类型
type StorageType string

const (
	LocalStorage StorageType = "local" // 本地存储
	S3Storage    StorageType = "s3"    // S3协议存储
)

// SystemConfig 系统配置
type SystemConfig struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ConfigKey   string    `json:"configKey" gorm:"type:varchar(100);not null;uniqueIndex"`           // 配置键
	ConfigValue string    `json:"configValue" gorm:"type:text;not null"`                             // 配置值
	Description string    `json:"description" gorm:"type:varchar(255);not null;default:''"`          // 配置描述
	CreatedAt   time.Time `json:"createdAt" gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP"` // 创建时间
	UpdatedAt   time.Time `json:"updatedAt" gorm:"type:datetime;not null"`                           // 更新时间
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// BackupTask 备份任务
type BackupTask struct {
	ID         int64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	Name       string                 `json:"name" gorm:"type:varchar(100);not null"`                            // 任务名称
	Type       BackupType             `json:"type" gorm:"type:varchar(20);not null"`                             // 备份类型
	SourceInfo string                 `json:"sourceInfo" gorm:"type:text;not null"`                              // 源信息，JSON格式，根据不同类型包含不同内容
	Schedule   string                 `json:"schedule" gorm:"type:varchar(100);not null"`                        // Cron表达式
	Enabled    bool                   `json:"enabled" gorm:"type:tinyint(1);not null;default:1"`                 // 是否启用
	CreatedAt  time.Time              `json:"createdAt" gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP"` // 创建时间
	UpdatedAt  time.Time              `json:"updatedAt" gorm:"type:datetime;not null"`                           // 更新时间
	ExtraData  map[string]interface{} `json:"extraData" gorm:"-"`                                                // 额外数据，不持久化到数据库
}

// TableName 指定表名
func (BackupTask) TableName() string {
	return "backup_tasks"
}

// DatabaseSourceInfo 数据库源信息
type DatabaseSourceInfo struct {
	Type     string `json:"type"`     // 数据库类型
	Host     string `json:"host"`     // 主机
	Port     int    `json:"port"`     // 端口
	User     string `json:"user"`     // 用户名
	Password string `json:"password"` // 密码
	Database string `json:"database"` // 数据库名，为空或"all"时表示备份所有数据库
}

// FileSourceInfo 文件源信息
type FileSourceInfo struct {
	Paths []string `json:"paths"` // 文件或目录路径
}

// BackupRecord 备份记录
type BackupRecord struct {
	ID            int64        `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID        int64        `json:"taskId" gorm:"not null;index"`                                        // 任务ID
	TaskName      string       `json:"taskName" gorm:"-"`                                                   // 任务名称（不映射到数据库）
	Status        BackupStatus `json:"status" gorm:"type:varchar(20);not null"`                             // 状态
	StartTime     time.Time    `json:"startTime" gorm:"type:datetime;not null"`                             // 开始时间
	EndTime       time.Time    `json:"endTime" gorm:"type:datetime;not null;default:'1970-01-01 00:00:00'"` // 结束时间
	FileSize      int64        `json:"fileSize" gorm:"not null;default:0"`                                  // 备份文件大小，单位字节
	FilePath      string       `json:"filePath" gorm:"type:varchar(255);not null;default:''"`               // 文件路径
	StorageType   StorageType  `json:"storageType" gorm:"type:varchar(20);not null;default:'local'"`        // 存储类型
	ErrorMessage  string       `json:"errorMessage" gorm:"type:text;not null"`                              // 错误信息
	BackupVersion string       `json:"backupVersion" gorm:"type:varchar(50);not null;default:''"`           // 备份版本
	CreatedAt     time.Time    `json:"createdAt" gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP"`   // 创建时间
	UpdatedAt     time.Time    `json:"updatedAt" gorm:"type:datetime;not null"`                             // 更新时间
}

// TableName 指定表名
func (BackupRecord) TableName() string {
	return "backup_records"
}
