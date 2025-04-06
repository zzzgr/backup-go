package repository

import (
	"backup-go/entity"
	"encoding/json"
	"errors"
	"time"
)

// BackupTaskRepository 备份任务仓库
type BackupTaskRepository struct {
	db interface{} // 使用空接口类型
}

// NewBackupTaskRepository 创建备份任务仓库
func NewBackupTaskRepository() *BackupTaskRepository {
	return &BackupTaskRepository{
		db: GetDB(),
	}
}

// Create 创建备份任务
func (r *BackupTaskRepository) Create(task *entity.BackupTask) error {
	// 确保时间戳字段正确设置
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = now
	}

	// 开始事务
	tx := GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 在事务中执行创建操作
	if err := tx.Create(task).Error; err != nil {
		tx.Rollback() // 发生错误时回滚
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// Update 更新备份任务
func (r *BackupTaskRepository) Update(task *entity.BackupTask) error {
	// 更新UpdatedAt字段
	task.UpdatedAt = time.Now()

	// 开始事务
	tx := GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 使用Map明确列出要更新的字段，确保零值也会被更新
	updateMap := map[string]interface{}{
		"name":        task.Name,
		"type":        task.Type,
		"source_info": task.SourceInfo,
		"schedule":    task.Schedule,
		"enabled":     task.Enabled, // 明确包含enabled字段
		"updated_at":  task.UpdatedAt,
	}

	// 在事务中执行更新操作
	if err := tx.Model(task).Where("id = ?", task.ID).Updates(updateMap).Error; err != nil {
		tx.Rollback() // 发生错误时回滚
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// Delete 删除备份任务
func (r *BackupTaskRepository) Delete(id int64) error {
	// 开始事务
	tx := GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 在事务中执行删除操作
	result := tx.Delete(&entity.BackupTask{}, id)
	if result.Error != nil {
		tx.Rollback() // 发生错误时回滚
		return result.Error
	}

	// 提交事务
	return tx.Commit().Error
}

// FindByID 根据ID查找备份任务
func (r *BackupTaskRepository) FindByID(id int64) (*entity.BackupTask, error) {
	var task entity.BackupTask
	result := GetDB().First(&task, id)

	if result.Error != nil {
		if errors.Is(result.Error, errors.New("record not found")) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &task, nil
}

// FindAll 查找所有备份任务
func (r *BackupTaskRepository) FindAll() ([]*entity.BackupTask, error) {
	var tasks []*entity.BackupTask

	result := GetDB().Order("id desc").Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

// GetEnabledTasks 获取所有启用的任务
func (r *BackupTaskRepository) GetEnabledTasks() ([]*entity.BackupTask, error) {
	var tasks []*entity.BackupTask

	result := GetDB().Where("enabled = ?", true).Order("id desc").Find(&tasks)
	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

// ParseDatabaseSourceInfo 解析数据库源信息
func (r *BackupTaskRepository) ParseDatabaseSourceInfo(task *entity.BackupTask) (*entity.DatabaseSourceInfo, error) {
	if task.Type != entity.DatabaseBackup {
		return nil, errors.New("task is not a database backup")
	}

	var info entity.DatabaseSourceInfo
	err := json.Unmarshal([]byte(task.SourceInfo), &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// ParseFileSourceInfo 解析文件源信息
func (r *BackupTaskRepository) ParseFileSourceInfo(task *entity.BackupTask) (*entity.FileSourceInfo, error) {
	if task.Type != entity.FileBackup {
		return nil, errors.New("task is not a file backup")
	}

	var info entity.FileSourceInfo
	err := json.Unmarshal([]byte(task.SourceInfo), &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// FindAllPaginated 分页查询所有备份任务
func (r *BackupTaskRepository) FindAllPaginated(page, pageSize int) ([]*entity.BackupTask, error) {
	var tasks []*entity.BackupTask

	offset := (page - 1) * pageSize

	result := GetDB().Order("id desc").
		Offset(offset).
		Limit(pageSize).
		Find(&tasks)

	if result.Error != nil {
		return nil, result.Error
	}

	return tasks, nil
}

// CountAll 计算备份任务总数
func (r *BackupTaskRepository) CountAll() (int64, error) {
	var count int64

	result := GetDB().Model(&entity.BackupTask{}).
		Count(&count)

	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}
