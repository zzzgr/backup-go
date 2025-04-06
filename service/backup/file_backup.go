package backup

import (
	"archive/zip"
	"backup-go/entity"
	"backup-go/repository"
	"backup-go/service/config"
	"backup-go/service/storage"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// FileBackupService 文件备份服务
type FileBackupService struct {
	taskRepo       *repository.BackupTaskRepository
	recordRepo     *repository.BackupRecordRepository
	storageType    entity.StorageType
	webhookService *config.WebhookService
}

// NewFileBackupService 创建文件备份服务
func NewFileBackupService() *FileBackupService {
	return &FileBackupService{
		taskRepo:       repository.NewBackupTaskRepository(),
		recordRepo:     repository.NewBackupRecordRepository(),
		storageType:    entity.LocalStorage, // 默认使用本地存储
		webhookService: config.NewWebhookService(),
	}
}

// Execute 执行备份
func (s *FileBackupService) Execute(task *entity.BackupTask) (*entity.BackupRecord, error) {
	// 解析源信息
	sourceInfo, err := s.taskRepo.ParseFileSourceInfo(task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file source info: %w", err)
	}

	// 创建备份记录
	record := &entity.BackupRecord{
		TaskID:    task.ID,
		Status:    entity.StatusRunning,
		StartTime: time.Now(),
	}
	err = s.recordRepo.Create(record)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup record: %w", err)
	}

	// 执行备份
	backupVersion := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("files_%s.zip", backupVersion)

	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "file_backup")
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建ZIP文件
	tempFilePath := filepath.Join(tempDir, filename)
	zipFile, err := os.Create(tempFilePath)
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to create zip file: %w", err)
	}

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 添加文件到ZIP
	for _, path := range sourceInfo.Paths {
		err = s.addFileToZip(zipWriter, path, "")
		if err != nil {
			zipWriter.Close()
			zipFile.Close()
			s.updateRecordStatus(record, entity.StatusFailed, err.Error())
			return record, fmt.Errorf("failed to add file to zip: %w", err)
		}
	}

	// 关闭ZIP writer
	err = zipWriter.Close()
	if err != nil {
		zipFile.Close()
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to close zip writer: %w", err)
	}

	// 关闭文件
	defer zipFile.Close()

	// 获取ZIP文件信息
	fileInfo, err := zipFile.Stat()
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to get file info: %w", err)
	}

	// 上传到存储
	storageService, err := storage.NewStorageService("")
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to create storage service: %w", err)
	}

	// 重新打开文件用于上传
	zipFile.Seek(0, 0)
	filePath, err := storageService.Save(filename, zipFile)
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to save backup file: %w", err)
	}

	// 更新记录
	record.Status = entity.StatusSuccess
	record.EndTime = time.Now()
	record.FileSize = fileInfo.Size()
	record.FilePath = filePath
	record.BackupVersion = backupVersion
	record.StorageType = storageService.GetStorageType()

	if err := s.recordRepo.Update(record); err != nil {
		return record, fmt.Errorf("failed to update backup record: %w", err)
	}

	// 发送备份成功通知
	task, tErr := s.taskRepo.FindByID(record.TaskID)
	if tErr == nil && task != nil {
		duration := record.EndTime.Sub(record.StartTime)
		// 尝试发送通知，忽略错误
		_ = s.webhookService.SendBackupSuccessNotification(
			task.Name,
			record.FileSize,
			record.FilePath,
			duration,
		)
	}

	return record, nil
}

// 添加文件到ZIP
func (s *FileBackupService) addFileToZip(zipWriter *zip.Writer, path, baseInZip string) error {
	// 获取文件信息
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// 构建ZIP中的路径
	var zipPath string
	if baseInZip == "" {
		zipPath = filepath.Base(path)
	} else {
		zipPath = filepath.Join(baseInZip, filepath.Base(path))
	}

	// 处理目录
	if info.IsDir() {
		// 为目录创建条目
		if zipPath != "" {
			_, err = zipWriter.Create(zipPath + "/")
			if err != nil {
				return err
			}
		}

		// 读取目录内容
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}

		// 递归处理子文件和子目录
		for _, file := range files {
			filePath := filepath.Join(path, file.Name())
			err = s.addFileToZip(zipWriter, filePath, zipPath)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// 处理普通文件
	fileToZip, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// 创建ZIP中的文件
	writer, err := zipWriter.Create(zipPath)
	if err != nil {
		return err
	}

	// 写入内容
	_, err = io.Copy(writer, fileToZip)
	if err != nil {
		return err
	}

	return nil
}

// GetBackupType 获取备份类型
func (s *FileBackupService) GetBackupType() entity.BackupType {
	return entity.FileBackup
}

// 更新记录状态
func (s *FileBackupService) updateRecordStatus(record *entity.BackupRecord, status entity.BackupStatus, errorMsg string) {
	record.Status = status
	record.EndTime = time.Now()
	record.ErrorMessage = errorMsg

	_ = s.recordRepo.Update(record)

	// 如果是失败状态，发送Webhook通知
	if status == entity.StatusFailed {
		task, err := s.taskRepo.FindByID(record.TaskID)
		if err == nil && task != nil {
			// 尝试发送通知，忽略错误
			_ = s.webhookService.SendBackupFailureNotification(
				task.Name,
				errorMsg,
			)
		}
	}
}
