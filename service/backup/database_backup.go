package backup

import (
	"backup-go/entity"
	"backup-go/repository"
	"backup-go/service/config"
	"backup-go/service/storage"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// DatabaseBackupService 数据库备份服务
type DatabaseBackupService struct {
	taskRepo       *repository.BackupTaskRepository
	recordRepo     *repository.BackupRecordRepository
	storageType    entity.StorageType
	webhookService *config.WebhookService
}

// NewDatabaseBackupService 创建数据库备份服务
func NewDatabaseBackupService() *DatabaseBackupService {
	return &DatabaseBackupService{
		taskRepo:       repository.NewBackupTaskRepository(),
		recordRepo:     repository.NewBackupRecordRepository(),
		storageType:    entity.LocalStorage, // 默认使用本地存储
		webhookService: config.NewWebhookService(),
	}
}

// Execute 执行备份
func (s *DatabaseBackupService) Execute(task *entity.BackupTask) (*entity.BackupRecord, error) {
	// 解析源信息
	sourceInfo, err := s.taskRepo.ParseDatabaseSourceInfo(task)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database source info: %w", err)
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

	// 生成文件名，添加任务ID和任务名称以避免重复
	// 为任务名称去除特殊字符，避免不合法的文件名
	safeName := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, task.Name)

	var filename string
	if sourceInfo.Database == "" || sourceInfo.Database == "all" {
		filename = fmt.Sprintf("task_%d_%s_all_databases_%s.sql", task.ID, safeName, backupVersion)
	} else {
		filename = fmt.Sprintf("task_%d_%s_%s_%s.sql", task.ID, safeName, sourceInfo.Database, backupVersion)
	}

	// 创建临时目录
	//tempDir, err := ioutil.TempDir("", "db_backup")
	tempDir, err := ioutil.TempDir("", "db_backup")
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempFilePath := filepath.Join(tempDir, filename)

	// 执行备份命令
	var cmd *exec.Cmd
	switch sourceInfo.Type {
	case "mysql":
		// 构造基本的mysqldump命令参数
		args := []string{
			"-h" + sourceInfo.Host,
			"-P" + fmt.Sprintf("%d", sourceInfo.Port),
			"-u" + sourceInfo.User,
			"-p" + sourceInfo.Password,
			"--result-file=" + tempFilePath,
			"--ssl-mode=DISABLED", // mysql
			//"--ssl=0", // 打包 mariadb
		}

		// 判断是否为全库备份（备份所有数据库）
		if sourceInfo.Database == "" || sourceInfo.Database == "all" {
			// 备份所有数据库
			args = append(args, "--all-databases")
		} else {
			// 备份指定的数据库
			args = append(args, "--databases", sourceInfo.Database)
		}

		// 用于调试，输出执行的命令
		cmdStr := "mysqldump "
		for _, arg := range args {
			cmdStr += arg + " "
		}
		log.Println(cmdStr)

		cmd = exec.Command("mysqldump", args...)
	default:
		err := fmt.Errorf("unsupported database type: %s", sourceInfo.Type)
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, err
	}

	// 执行命令
	if err := cmd.Run(); err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("backup command failed: %w", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(tempFilePath)
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to get file info: %w", err)
	}

	// 读取备份文件
	backupData, err := os.Open(tempFilePath)
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to read backup file: %w", err)
	}
	defer backupData.Close()

	// 上传到存储
	storageService, err := storage.NewStorageService("")
	if err != nil {
		s.updateRecordStatus(record, entity.StatusFailed, err.Error())
		return record, fmt.Errorf("failed to create storage service: %w", err)
	}

	filePath, err := storageService.Save(filename, backupData)
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

// GetBackupType 获取备份类型
func (s *DatabaseBackupService) GetBackupType() entity.BackupType {
	return entity.DatabaseBackup
}

// 更新记录状态
func (s *DatabaseBackupService) updateRecordStatus(record *entity.BackupRecord, status entity.BackupStatus, errorMsg string) {
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
