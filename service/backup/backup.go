package backup

import (
	"backup-go/entity"
)

// BackupService 备份服务接口
type BackupService interface {
	// Execute 执行备份
	Execute(task *entity.BackupTask) (*entity.BackupRecord, error)

	// GetBackupType 获取备份类型
	GetBackupType() entity.BackupType
}

// 创建备份服务
func NewBackupService(backupType entity.BackupType) (BackupService, error) {
	switch backupType {
	case entity.DatabaseBackup:
		return NewDatabaseBackupService(), nil
	case entity.FileBackup:
		return NewFileBackupService(), nil
	default:
		return nil, nil
	}
}
