package controller

import (
	"backup-go/entity"
	"backup-go/model"
	"backup-go/repository"
	configService "backup-go/service/config"
	"backup-go/service/storage"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// RecordController 备份记录控制器
type RecordController struct {
	recordRepo *repository.BackupRecordRepository
	taskRepo   *repository.BackupTaskRepository
}

// NewRecordController 创建备份记录控制器
func NewRecordController() *RecordController {
	return &RecordController{
		recordRepo: repository.NewBackupRecordRepository(),
		taskRepo:   repository.NewBackupTaskRepository(),
	}
}

// GetRecord 获取备份记录
func (c *RecordController) GetRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "Invalid record ID"))
		return
	}

	record, err := c.recordRepo.FindByID(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to find record: "+err.Error()))
		return
	}
	if record == nil {
		c.writeJSON(w, model.Error(404, "Record not found"))
		return
	}

	// 添加任务名称
	task, err := c.taskRepo.FindByID(record.TaskID)
	if err == nil && task != nil {
		record.TaskName = task.Name
	} else {
		record.TaskName = "未知任务"
	}

	c.writeJSON(w, model.Success(record))
}

// GetAllRecords 获取所有备份记录
func (c *RecordController) GetAllRecords(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	records, err := c.recordRepo.FindAll(page, pageSize)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to find records: "+err.Error()))
		return
	}

	count, err := c.recordRepo.CountAll()
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to count records: "+err.Error()))
		return
	}

	// 为每条记录添加任务名称
	for _, record := range records {
		// 根据TaskID查询任务名称
		task, err := c.taskRepo.FindByID(record.TaskID)
		if err == nil && task != nil {
			// 将任务名称添加到记录中
			record.TaskName = task.Name
		} else {
			// 如果查询失败，显示未知任务
			record.TaskName = "未知任务"
		}
	}

	result := map[string]interface{}{
		"records":  records,
		"total":    count,
		"page":     page,
		"pageSize": pageSize,
	}

	c.writeJSON(w, model.Success(result))
}

// GetTaskRecords 获取任务的备份记录
func (c *RecordController) GetTaskRecords(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(r.URL.Query().Get("taskId"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "Invalid task ID"))
		return
	}

	// 支持分页参数
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// 获取任务的总记录数
	count, err := c.recordRepo.CountByTaskID(taskID)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to count records: "+err.Error()))
		return
	}

	// 获取带分页的记录
	records, err := c.recordRepo.FindByTaskIDPaginated(taskID, page, pageSize)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to find records: "+err.Error()))
		return
	}

	// 获取任务信息以添加任务名称
	task, err := c.taskRepo.FindByID(taskID)
	if err == nil && task != nil {
		// 为所有记录添加任务名称
		for _, record := range records {
			record.TaskName = task.Name
		}
	} else {
		// 如果查询任务失败，使用"未知任务"作为任务名称
		for _, record := range records {
			record.TaskName = "未知任务"
		}
	}

	// 构建分页结果
	result := map[string]interface{}{
		"records":  records,
		"total":    count,
		"page":     page,
		"pageSize": pageSize,
	}

	c.writeJSON(w, model.Success(result))
}

// DownloadBackup 下载备份文件
func (c *RecordController) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	// 检查是否启用密码保护
	cs := configService.NewConfigService()
	passwordConfig, err := cs.GetConfigByKey("system.password")

	// 如果设置了密码，需要验证token
	if err == nil && passwordConfig.ConfigValue != "" {
		// 获取token，可以从多个位置获取
		var token string

		// 1. 从Authorization头获取
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 2. 从URL查询参数获取
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		// 3. 从POST表单获取
		if token == "" {
			err := r.ParseForm()
			if err == nil {
				token = r.FormValue("token")
			}
		}

		// 验证token
		if token == "" || !IsValidToken(token) {
			c.writeJSON(w, model.Error(401, "未授权访问，请先登录"))
			return
		}
	}

	// 直接通过文件路径下载
	path := r.URL.Query().Get("path")
	if path != "" {
		// 检查路径安全性，避免任意文件访问
		if strings.Contains(path, "..") {
			c.writeJSON(w, model.Error(403, "Invalid file path"))
			return
		}

		// 获取存储服务
		storageType := entity.LocalStorage // 默认本地存储

		// 根据路径前缀判断存储类型
		if strings.HasPrefix(path, "s3://") {
			storageType = entity.S3Storage
		}

		// 获取对应的存储服务
		storageService, err := storage.NewStorageService(storageType)
		if err != nil {
			c.writeJSON(w, model.Error(500, "Failed to create storage service: "+err.Error()))
			return
		}

		// 获取文件
		file, err := storageService.Get(path)
		if err != nil {
			c.writeJSON(w, model.Error(500, "Failed to get backup file: "+err.Error()))
			return
		}
		defer file.Close()

		// 从文件路径中提取文件名
		filename := filepath.Base(path)

		// 设置响应头
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", strconv.Quote(filename)))
		w.Header().Set("Content-Type", "application/octet-stream")

		// 发送文件内容
		_, err = io.Copy(w, file)
		if err != nil {
			c.writeJSON(w, model.Error(500, "Failed to send file: "+err.Error()))
			return
		}

		return
	}

	// 通过记录ID下载
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "Invalid record ID"))
		return
	}

	// 获取备份记录
	record, err := c.recordRepo.FindByID(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to find record: "+err.Error()))
		return
	}
	if record == nil {
		c.writeJSON(w, model.Error(404, "Record not found"))
		return
	}

	// 获取任务信息
	task, err := c.taskRepo.FindByID(record.TaskID)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to find task: "+err.Error()))
		return
	}
	if task == nil {
		c.writeJSON(w, model.Error(404, "Task not found"))
		return
	}

	// 获取存储服务
	// 优先使用记录中存储的存储类型
	storageType := record.StorageType

	// 如果记录中没有存储类型（旧数据兼容处理），使用系统配置
	if storageType == "" {
		// 从系统配置表读取存储类型
		cs := configService.NewConfigService()
		storageTypeStr, err := cs.GetConfigValue("storage.type")
		if err == nil && storageTypeStr != "" {
			storageType = entity.StorageType(storageTypeStr)
		} else {
			// 配置表中也没有，尝试从文件路径判断
			filePath := record.FilePath
			if strings.HasPrefix(filePath, "s3://") || strings.HasPrefix(filePath, "backups/") {
				// S3存储或特定格式
				storageType = entity.S3Storage
			} else if filepath.IsAbs(filePath) || strings.HasPrefix(filePath, "./") || strings.HasPrefix(filePath, "../") {
				// 绝对路径或相对路径，视为本地存储
				storageType = entity.LocalStorage
			}
		}
	}

	// 获取对应的存储服务
	storageService, err := storage.NewStorageService(storageType)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to create storage service: "+err.Error()))
		return
	}

	// 获取文件
	file, err := storageService.Get(record.FilePath)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to get backup file: "+err.Error()))
		return
	}
	defer file.Close()

	// 从文件路径中提取文件名
	filename := filepath.Base(record.FilePath)

	// 设置响应头
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", strconv.Quote(filename)))
	w.Header().Set("Content-Type", "application/octet-stream")

	// 发送文件内容
	_, err = io.Copy(w, file)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to send file: "+err.Error()))
		return
	}
}

// DeleteRecord 删除备份记录
func (c *RecordController) DeleteRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "无效的记录ID"))
		return
	}

	// 获取备份记录
	record, err := c.recordRepo.FindByID(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "查找记录失败: "+err.Error()))
		return
	}
	if record == nil {
		c.writeJSON(w, model.Error(404, "记录不存在"))
		return
	}

	// 如果有备份文件路径，尝试删除文件
	if record.FilePath != "" {
		// 获取存储服务
		// 优先使用记录中存储的存储类型
		storageType := record.StorageType

		// 如果记录中没有存储类型（旧数据兼容处理），尝试从文件路径判断
		if storageType == "" {
			filePath := record.FilePath
			if strings.HasPrefix(filePath, "s3://") {
				storageType = entity.S3Storage
			} else {
				storageType = entity.LocalStorage
			}
		}

		// 获取对应的存储服务
		storageService, err := storage.NewStorageService(storageType)
		if err != nil {
			c.writeJSON(w, model.Error(500, "创建存储服务失败: "+err.Error()))
			return
		}

		// 删除文件
		err = storageService.Delete(record.FilePath)
		if err != nil {
			// 仅记录日志，不中断流程
			log.Printf("删除文件失败: %s, 错误: %v", record.FilePath, err)
		}
	}

	// 删除数据库记录
	err = c.recordRepo.Delete(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "删除记录失败: "+err.Error()))
		return
	}

	c.writeJSON(w, model.Success(nil))
}

// 写入JSON响应
func (c *RecordController) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
