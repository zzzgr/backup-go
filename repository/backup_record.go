package repository

import (
	"backup-go/entity"
	"errors"
	"fmt"
	"time"
)

// BackupRecordRepository 备份记录仓库
type BackupRecordRepository struct {
	db interface{} // 使用空接口类型
}

// NewBackupRecordRepository 创建备份记录仓库
func NewBackupRecordRepository() *BackupRecordRepository {
	return &BackupRecordRepository{
		db: GetDB(),
	}
}

// Create 创建备份记录
func (r *BackupRecordRepository) Create(record *entity.BackupRecord) error {
	// 确保时间戳字段正确设置
	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = now
	}

	// 开始事务
	tx := GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 在事务中执行创建操作
	if err := tx.Create(record).Error; err != nil {
		tx.Rollback() // 发生错误时回滚
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// Update 更新备份记录
func (r *BackupRecordRepository) Update(record *entity.BackupRecord) error {
	// 更新UpdatedAt字段
	record.UpdatedAt = time.Now()

	// 开始事务
	tx := GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 在事务中执行更新操作
	if err := tx.Model(record).Updates(record).Error; err != nil {
		tx.Rollback() // 发生错误时回滚
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

// FindByID 根据ID查找备份记录
func (r *BackupRecordRepository) FindByID(id int64) (*entity.BackupRecord, error) {
	var record entity.BackupRecord
	result := GetDB().First(&record, id)

	if result.Error != nil {
		if errors.Is(result.Error, errors.New("record not found")) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &record, nil
}

// FindByTaskID 根据任务ID查找备份记录
func (r *BackupRecordRepository) FindByTaskID(taskID int64) ([]*entity.BackupRecord, error) {
	var records []*entity.BackupRecord

	result := GetDB().Where("task_id = ?", taskID).Order("start_time desc").Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}

	return records, nil
}

// FindLatestByTaskID 获取任务最新的备份记录
func (r *BackupRecordRepository) FindLatestByTaskID(taskID int64) (*entity.BackupRecord, error) {
	var record entity.BackupRecord

	result := GetDB().Where("task_id = ?", taskID).Order("start_time desc").First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, errors.New("record not found")) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &record, nil
}

// FindAll 查询所有备份记录，支持分页
func (r *BackupRecordRepository) FindAll(page, pageSize int) ([]*entity.BackupRecord, error) {
	var records []*entity.BackupRecord

	offset := (page - 1) * pageSize

	result := GetDB().Order("start_time desc").Offset(offset).Limit(pageSize).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}

	return records, nil
}

// CountAll 统计所有备份记录数量
func (r *BackupRecordRepository) CountAll() (int64, error) {
	var count int64

	result := GetDB().Model(&entity.BackupRecord{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

// FindByStatus 根据状态查找备份记录
func (r *BackupRecordRepository) FindByStatus(status entity.BackupStatus) ([]*entity.BackupRecord, error) {
	var records []*entity.BackupRecord

	result := GetDB().Where("status = ?", status).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}

	return records, nil
}

// Delete 删除备份记录
func (r *BackupRecordRepository) Delete(id int64) error {
	// 开始事务
	tx := GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 在事务中执行删除操作
	result := tx.Delete(&entity.BackupRecord{}, id)
	if result.Error != nil {
		tx.Rollback() // 发生错误时回滚
		return result.Error
	}

	// 提交事务
	return tx.Commit().Error
}

// DeleteByTaskID 根据任务ID删除所有相关备份记录
func (r *BackupRecordRepository) DeleteByTaskID(taskID int64) error {
	// 开始事务
	tx := GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 在事务中执行删除操作
	result := tx.Where("task_id = ?", taskID).Delete(&entity.BackupRecord{})
	if result.Error != nil {
		tx.Rollback() // 发生错误时回滚
		return result.Error
	}

	// 提交事务
	return tx.Commit().Error
}

// FindByTaskIDPaginated 分页查询指定任务的备份记录
func (r *BackupRecordRepository) FindByTaskIDPaginated(taskID int64, page, pageSize int) ([]*entity.BackupRecord, error) {
	var records []*entity.BackupRecord

	offset := (page - 1) * pageSize

	result := GetDB().Where("task_id = ?", taskID).
		Order("start_time desc").
		Offset(offset).
		Limit(pageSize).
		Find(&records)

	if result.Error != nil {
		return nil, result.Error
	}

	return records, nil
}

// CountByTaskID 计算指定任务的备份记录总数
func (r *BackupRecordRepository) CountByTaskID(taskID int64) (int64, error) {
	var count int64

	result := GetDB().Model(&entity.BackupRecord{}).
		Where("task_id = ?", taskID).
		Count(&count)

	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

// FindOlderThan 查找早于指定日期的记录
func (r *BackupRecordRepository) FindOlderThan(date time.Time) ([]*entity.BackupRecord, error) {
	var records []*entity.BackupRecord

	// 使用时间条件查询记录
	result := GetDB().Where("start_time <= ?", date).
		Where("file_path != ?", "").                // 只查找有文件路径的记录
		Where("status != ?", entity.StatusCleaned). // 排除已清理的记录
		Find(&records)

	if result.Error != nil {
		return nil, fmt.Errorf("查询早于%s的记录失败: %w", date.Format("2006-01-02"), result.Error)
	}

	return records, nil
}
