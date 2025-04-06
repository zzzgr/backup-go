package controller

import (
	"backup-go/model"
	"backup-go/service/cleanup"
	"log"
	"net/http"
	"strconv"
)

// CleanupController 清理控制器
type CleanupController struct {
	cleanupService *cleanup.CleanupService
}

// NewCleanupController 创建清理控制器
func NewCleanupController() *CleanupController {
	return &CleanupController{
		cleanupService: cleanup.GetCleanupService(),
	}
}

// ExecuteCleanup 执行清理
func (c *CleanupController) ExecuteCleanup(w http.ResponseWriter, r *http.Request) {
	// 执行清理并获取结果
	result := c.cleanupService.ExecuteAndGetResult()

	// 发送webhook通知(手动执行)
	if err := c.cleanupService.GetWebhookService().SendCleanupNotification(result.Success, result.Failed, result.Skipped, false, result.ErrorMessages); err != nil {
		// 仅记录日志，不影响正常响应
		log.Printf("发送清理通知失败: %v", err)
	}

	// 构建响应消息
	message := ""
	if len(result.ErrorMessages) > 0 {
		message = "清理任务完成，但存在以下问题: " + result.ErrorMessages[0]
		if len(result.ErrorMessages) > 1 {
			message += " (还有" + strconv.Itoa(len(result.ErrorMessages)-1) + "个其他错误)"
		}
	} else {
		message = "清理任务已完成"
	}

	// 返回响应
	data := map[string]interface{}{
		"success": result.Success,
		"failed":  result.Failed,
		"skipped": result.Skipped,
		"errors":  result.ErrorMessages,
	}

	c.writeJSON(w, model.SuccessWithMsg(data, message))
}

// writeJSON 输出JSON
func (c *CleanupController) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	model.WriteJSON(w, data)
}
