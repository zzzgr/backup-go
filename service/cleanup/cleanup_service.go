package cleanup

import (
	"backup-go/entity"
	"backup-go/repository"
	"backup-go/service/config"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/robfig/cron/v3"
)

// CleanupService 清理服务
type CleanupService struct {
	cron           *cron.Cron
	configService  *config.ConfigService
	recordRepo     *repository.BackupRecordRepository
	webhookService *config.WebhookService
	cronEntryID    cron.EntryID
	mutex          sync.Mutex
	running        bool
}

var (
	instance *CleanupService
	once     sync.Once
)

// GetCleanupService 获取清理服务单例
func GetCleanupService() *CleanupService {
	once.Do(func() {
		instance = &CleanupService{
			cron:           cron.New(cron.WithSeconds()),
			configService:  config.NewConfigService(),
			recordRepo:     repository.NewBackupRecordRepository(),
			webhookService: config.NewWebhookService(),
		}
	})
	return instance
}

// Start 启动清理服务
func (s *CleanupService) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		log.Println("清理服务已经在运行")
		return
	}

	// 设置每天凌晨2点运行清理任务
	var err error
	s.cronEntryID, err = s.cron.AddFunc("0 0 2 * * *", s.cleanup)
	if err != nil {
		log.Printf("添加清理任务失败: %v", err)
		return
	}

	// 启动cron
	s.cron.Start()
	s.running = true
	log.Println("清理服务启动成功，将在每天凌晨2点执行清理任务")
}

// Stop 停止清理服务
func (s *CleanupService) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	log.Println("清理服务已停止")
}

// ExecuteNow 立即执行清理
func (s *CleanupService) ExecuteNow() {
	go s.cleanup()
}

// ExecuteAndGetResult 执行清理并返回结果
func (s *CleanupService) ExecuteAndGetResult() *CleanupResult {
	log.Println("开始执行清理任务...")
	result := &CleanupResult{
		ErrorMessages: []string{},
	}

	// 获取清理天数配置
	daysStr, err := s.configService.GetConfigValue("system.autoCleanupDays")
	if err != nil {
		errMsg := "获取清理天数配置失败: " + err.Error()
		log.Println(errMsg)
		result.ErrorMessages = append(result.ErrorMessages, errMsg)
		return result
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil {
		errMsg := "清理天数配置值无效: " + err.Error()
		log.Println(errMsg)
		result.ErrorMessages = append(result.ErrorMessages, errMsg)
		return result
	}

	// 0表示不清理
	if days <= 0 {
		log.Println("清理天数设置为0，跳过清理")
		result.ErrorMessages = append(result.ErrorMessages, "清理天数设置为0，跳过清理")
		return result
	}

	// 计算清理日期
	cleanupDate := time.Now().AddDate(0, 0, -days)
	log.Printf("将清理%d天前（%s）之前的备份文件", days, cleanupDate.Format("2006-01-02"))

	// 获取所有需要清理的记录
	records, err := s.recordRepo.FindOlderThan(cleanupDate)
	if err != nil {
		errMsg := "查询过期备份记录失败: " + err.Error()
		log.Println(errMsg)
		result.ErrorMessages = append(result.ErrorMessages, errMsg)
		return result
	}

	log.Printf("找到%d条需要清理的备份记录", len(records))
	if len(records) == 0 {
		return result
	}

	// 获取存储配置（这些只用于本地存储路径）
	localPath, _ := s.configService.GetConfigValue("storage.localPath")
	if localPath == "" {
		localPath = "./backup" // 默认路径
	}

	// 获取S3配置信息
	s3Endpoint, _ := s.configService.GetConfigValue("storage.s3Endpoint")
	s3Region, _ := s.configService.GetConfigValue("storage.s3Region")
	s3AccessKey, _ := s.configService.GetConfigValue("storage.s3AccessKey")
	s3SecretKey, _ := s.configService.GetConfigValue("storage.s3SecretKey")
	s3Bucket, _ := s.configService.GetConfigValue("storage.s3Bucket")

	// 清理记录
	for _, record := range records {
		// 根据记录中的StorageType来处理不同的存储类型
		switch record.StorageType {
		case entity.LocalStorage:
			// 获取本地存储路径配置
			localPath, _ := s.configService.GetConfigValue("storage.localPath")
			if localPath == "" {
				localPath = "backups"
			}
			// 本地存储文件清理
			if s.cleanupLocalFile(record, localPath) {
				result.Success++
			} else {
				result.Failed++
			}
		case entity.S3Storage:
			// 检查S3配置是否有效
			if s3Endpoint == "" || s3AccessKey == "" || s3SecretKey == "" || s3Bucket == "" {
				errMsg := "S3配置不完整，跳过S3文件清理: " + record.FilePath
				log.Println(errMsg)
				result.Skipped++
				result.ErrorMessages = append(result.ErrorMessages, errMsg)
				continue
			}
			// S3存储文件清理
			if s.cleanupS3File(record, s3Endpoint, s3Region, s3AccessKey, s3SecretKey, s3Bucket) {
				result.Success++
			} else {
				result.Failed++
			}
		default:
			errMsg := "未知的存储类型: " + string(record.StorageType)
			log.Println(errMsg)
			result.Skipped++
			result.ErrorMessages = append(result.ErrorMessages, errMsg)
			continue
		}
	}

	log.Printf("清理任务完成。成功: %d, 失败: %d, 跳过: %d", result.Success, result.Failed, result.Skipped)
	return result
}

// cleanup 执行清理任务
func (s *CleanupService) cleanup() {
	result := s.ExecuteAndGetResult()
	log.Printf("清理任务完成。成功: %d, 失败: %d, 跳过: %d", result.Success, result.Failed, result.Skipped)
	for _, errMsg := range result.ErrorMessages {
		log.Println("清理错误: " + errMsg)
	}

	// 发送webhook通知
	if err := s.webhookService.SendCleanupNotification(result.Success, result.Failed, result.Skipped, true, result.ErrorMessages); err != nil {
		log.Printf("发送清理通知失败: %v", err)
	}
}

// cleanupLocalFile 清理本地文件
func (s *CleanupService) cleanupLocalFile(record *entity.BackupRecord, localPath string) bool {
	if record.FilePath == "" {
		log.Printf("记录 %d 没有文件路径，跳过", record.ID)
		return true
	}

	// 本地存储，直接删除文件
	filePath := record.FilePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(localPath, filePath)
	}

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("文件已不存在: %s", filePath)
			// 更新数据库记录
			record.FilePath = ""
			record.FileSize = 0
			record.Status = "cleaned" // 标记为已清理状态
			record.ErrorMessage = "文件已被自动清理"
			if err := s.recordRepo.Update(record); err != nil {
				log.Printf("更新记录失败: %v", err)
				return false
			}
			return true
		}
		log.Printf("删除本地文件失败: %s, 错误: %v", filePath, err)
		return false
	}

	// 更新数据库记录
	record.FilePath = ""
	record.FileSize = 0
	record.Status = "cleaned" // 标记为已清理状态
	record.ErrorMessage = "文件已被自动清理"
	if err := s.recordRepo.Update(record); err != nil {
		log.Printf("更新记录失败: %v", err)
		return false
	}
	return true
}

// cleanupS3File 清理S3文件
func (s *CleanupService) cleanupS3File(record *entity.BackupRecord, endpoint, region, accessKey, secretKey, bucket string) bool {
	if record.FilePath == "" {
		log.Printf("记录 %d 没有文件路径，跳过", record.ID)
		return true
	}

	// 数据库中存储的就是相对路径，直接用作S3对象键
	s3Key := record.FilePath
	log.Printf("使用S3对象键: %s", s3Key)

	// 使用AWS SDK删除S3文件
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})

	if err != nil {
		log.Printf("创建S3会话失败: %v", err)
		return false
	}

	// 创建S3客户端
	svc := s3.New(sess)

	// 删除S3对象
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
	}

	_, err = svc.DeleteObject(input)
	if err != nil {
		log.Printf("删除S3文件失败: %s, 错误: %v", s3Key, err)
		return false
	}

	log.Printf("成功删除S3文件: %s", s3Key)

	// 更新数据库记录
	record.FilePath = ""
	record.FileSize = 0
	record.Status = "cleaned" // 标记为已清理状态
	record.ErrorMessage = "文件已被自动清理"
	if err := s.recordRepo.Update(record); err != nil {
		log.Printf("更新记录失败: %v", err)
		return false
	}
	return true
}

// CleanupResult 清理结果
type CleanupResult struct {
	Success       int
	Failed        int
	Skipped       int
	ErrorMessages []string
}

// GetWebhookService 获取webhook服务
func (s *CleanupService) GetWebhookService() *config.WebhookService {
	return s.webhookService
}

// SendCleanupNotification 发送清理完成通知
func (s *CleanupService) SendCleanupNotification(success, failed, skipped int, isAuto bool, errorMessages []string) error {
	return s.webhookService.SendCleanupNotification(success, failed, skipped, isAuto, errorMessages)
}
