package config

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// WebhookService webhook服务
type WebhookService struct {
	configService *ConfigService
}

// WebhookData webhook数据
type WebhookData struct {
	TaskName    string `json:"taskName,omitempty"`
	Event       string `json:"event,omitempty"`
	Message     string `json:"message,omitempty"`
	TestMessage string `json:"testMessage,omitempty"`
}

// NewWebhookService 创建webhook服务
func NewWebhookService() *WebhookService {
	return &WebhookService{
		configService: NewConfigService(),
	}
}

// SendBackupFailureNotification 发送备份失败通知
func (w *WebhookService) SendBackupFailureNotification(taskName, errorMessage string) error {
	// 检查是否启用了webhook
	enabled, err := w.configService.GetConfigValue("webhook.enabled")
	if err != nil || enabled != "true" {
		return nil // 未启用或查询错误，不发送通知
	}

	// 准备数据
	data := &WebhookData{
		TaskName: taskName,
		Event:    "备份失败",
		Message:  errorMessage,
	}

	// 发送通知
	return w.sendWebhook(data)
}

// SendBackupSuccessNotification 发送备份成功通知
func (w *WebhookService) SendBackupSuccessNotification(taskName string, fileSize int64, filePath string, duration time.Duration) error {
	// 检查是否启用了webhook
	enabled, err := w.configService.GetConfigValue("webhook.enabled")
	if err != nil || enabled != "true" {
		return nil // 未启用或查询错误，不发送通知
	}

	// 格式化文件大小
	fileSizeStr := "0 B"
	if fileSize > 0 {
		// 简单格式化文件大小
		units := []string{"B", "KB", "MB", "GB", "TB"}
		size := float64(fileSize)
		unitIndex := 0
		for size >= 1024 && unitIndex < len(units)-1 {
			size /= 1024
			unitIndex++
		}
		fileSizeStr = fmt.Sprintf("%.2f %s", size, units[unitIndex])
	}

	// 格式化持续时间
	durationStr := fmt.Sprintf("%.2f秒", duration.Seconds())
	if duration.Minutes() >= 1 {
		durationStr = fmt.Sprintf("%.2f分钟", duration.Minutes())
	}

	// 准备消息内容
	message := fmt.Sprintf("备份完成，文件大小: %s，耗时: %s", fileSizeStr, durationStr)
	if filePath != "" {
		message += fmt.Sprintf("，文件路径: %s", filePath)
	}

	// 准备数据
	data := &WebhookData{
		TaskName: taskName,
		Event:    "备份成功",
		Message:  message,
	}

	// 发送通知
	return w.sendWebhook(data)
}

// SendCleanupNotification 发送清理操作完成通知
func (w *WebhookService) SendCleanupNotification(success, failed, skipped int, isAuto bool, errorMessages []string) error {
	// 检查是否启用了webhook
	enabled, err := w.configService.GetConfigValue("webhook.enabled")
	if err != nil || enabled != "true" {
		return nil // 未启用或查询错误，不发送通知
	}

	// 准备消息内容
	messageType := "自动"
	if !isAuto {
		messageType = "手动"
	}

	message := fmt.Sprintf("%s清理操作已完成。成功: %d, 失败: %d, 跳过: %d", messageType, success, failed, skipped)

	// 如果有错误消息，添加到通知中
	if len(errorMessages) > 0 {
		message += fmt.Sprintf("。发生了%d个错误，第一个错误: %s", len(errorMessages), errorMessages[0])
	}

	// 准备数据
	data := &WebhookData{
		TaskName: "系统清理",
		Event:    "清理完成",
		Message:  message,
	}

	// 发送通知
	return w.sendWebhook(data)
}

// TestWebhook 测试webhook
func (w *WebhookService) TestWebhook() error {
	// 检查是否启用了webhook
	enabled, err := w.configService.GetConfigValue("webhook.enabled")
	if err != nil || enabled != "true" {
		return fmt.Errorf("webhook未启用")
	}

	// 准备测试数据
	data := &WebhookData{
		TaskName:    "测试任务",
		Event:       "测试事件",
		Message:     "这是一条测试消息",
		TestMessage: "这是一条Webhook测试消息，如果您收到了，表示配置正确。",
	}

	// 从配置中获取URL等信息
	url, err := w.configService.GetConfigValue("webhook.url")
	if err != nil || url == "" {
		return fmt.Errorf("webhook URL未配置")
	}

	headersStr, _ := w.configService.GetConfigValue("webhook.headers")
	bodyTemplate, _ := w.configService.GetConfigValue("webhook.body")

	// 发送测试通知
	return w.sendWebhookWithConfig(data, url, headersStr, bodyTemplate)
}

// TestWebhookWithConfig 使用临时配置测试webhook
func (w *WebhookService) TestWebhookWithConfig(url, headersStr, bodyTemplate string) error {
	// URL已在控制器层验证，这里直接使用
	log.Printf("测试Webhook, URL: %s", url)

	// 准备测试数据
	data := &WebhookData{
		TaskName:    "测试任务",
		Event:       "测试事件",
		Message:     "这是一条测试消息",
		TestMessage: "这是一条Webhook测试消息，如果您收到了，表示配置正确。",
	}

	// 对URL进行变量替换
	finalURL := w.replaceVariables(url, data)

	// 检测是否包含特定的URL模式（bot接口+content参数）
	if strings.Contains(finalURL, "/bot/") && strings.Contains(finalURL, "content=") {
		log.Printf("检测到特殊的bot通知URL，使用专用方法处理")
		return w.handleChineseInURL(finalURL, data)
	}

	// 使用临时配置发送测试通知
	return w.sendWebhookWithConfig(data, url, headersStr, bodyTemplate)
}

// sendWebhook 发送webhook通知，使用已保存的配置
func (w *WebhookService) sendWebhook(data *WebhookData) error {
	// 获取webhook配置
	url, err := w.configService.GetConfigValue("webhook.url")
	if err != nil || url == "" {
		return fmt.Errorf("webhook URL未配置")
	}

	headersStr, _ := w.configService.GetConfigValue("webhook.headers")
	bodyTemplate, _ := w.configService.GetConfigValue("webhook.body")

	return w.sendWebhookWithConfig(data, url, headersStr, bodyTemplate)
}

// sendWebhookWithConfig 使用指定配置发送webhook通知
func (w *WebhookService) sendWebhookWithConfig(data *WebhookData, urlStr, headersStr, bodyTemplate string) error {
	// 替换变量
	body := w.replaceVariables(bodyTemplate, data)

	// 对URL也进行变量替换
	urlStr = w.replaceVariables(urlStr, data)

	// 额外处理URL，移除可能导致解析失败的字符
	urlStr = w.sanitizeURL(urlStr)

	// 确保URL编码正确
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("URL格式不正确: %w", err)
	}

	// 特殊处理URL参数，确保中文正确编码
	if parsedURL.RawQuery != "" {
		values := parsedURL.Query()
		parsedURL.RawQuery = values.Encode()
	}

	// 创建resty客户端
	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(500 * time.Millisecond)

	// 打印调试信息
	finalURL := parsedURL.String()
	log.Printf("发送webhook请求到: %s", finalURL)

	// 创建请求
	request := client.R()

	// 添加自定义请求头
	if headersStr != "" {
		headers := w.parseHeadersWithVariables(headersStr, data)
		request.SetHeaders(headers)
	}

	// 根据body决定是GET还是POST请求
	var resp *resty.Response

	if body == "" {
		// 如果请求体为空，则使用GET请求
		resp, err = request.Get(finalURL)
	} else {
		// 如果请求体不为空，则使用POST请求
		// 如果没有显式设置Content-Type，默认设置为application/json
		if headersStr == "" || !strings.Contains(strings.ToLower(headersStr), "content-type:") {
			request.SetHeader("Content-Type", "application/json")
		}
		resp, err = request.SetBody(body).Post(finalURL)
	}

	if err != nil {
		log.Printf("发送webhook失败: %s \n", err.Error())
		return fmt.Errorf("发送webhook失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode() != 200 {
		return fmt.Errorf("webhook服务返回错误状态码: %d, 响应体: %s", resp.StatusCode(), resp.String())
	}

	log.Printf("webhook请求成功，状态码: %d", resp.StatusCode())
	return nil
}

// replaceVariables 替换webhook模板中的变量
func (w *WebhookService) replaceVariables(template string, data *WebhookData) string {
	if template == "" {
		return ""
	}

	result := template

	// 对message内容进行特殊处理，确保其中不包含会导致URL解析失败的特殊字符
	safeMessage := ""
	if data.Message != "" {
		// 替换换行符和其他可能导致URL解析失败的控制字符
		safeMessage = strings.Map(func(r rune) rune {
			// 替换换行符、回车符、制表符等控制字符
			if r < 32 || r == 127 {
				return ' '
			}
			// 替换引号和反斜杠
			if r == '"' || r == '\\' || r == '\'' {
				return ' '
			}
			return r
		}, data.Message)
	}

	result = strings.ReplaceAll(result, "${taskName}", data.TaskName)
	result = strings.ReplaceAll(result, "${event}", data.Event)
	result = strings.ReplaceAll(result, "${message}", safeMessage)

	// 只有测试消息才使用这个字段
	if data.TestMessage != "" {
		// 同样处理TestMessage中的特殊字符
		safeTestMessage := strings.Map(func(r rune) rune {
			if r < 32 || r == 127 || r == '"' || r == '\\' || r == '\'' {
				return ' '
			}
			return r
		}, data.TestMessage)
		result = strings.ReplaceAll(result, "${testMessage}", safeTestMessage)
	}

	return result
}

// parseHeaders 将头部字符串解析为map
func (w *WebhookService) parseHeaders(headersStr string) map[string]string {
	headers := make(map[string]string)

	if headersStr == "" {
		return headers
	}

	// 解析每一行的header
	lines := strings.Split(headersStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 分割header名和值
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		headerName := strings.TrimSpace(parts[0])
		headerValue := strings.TrimSpace(parts[1])
		if headerName != "" && headerValue != "" {
			headers[headerName] = headerValue
		}
	}

	return headers
}

// parseHeadersWithVariables 将头部字符串解析为map，并替换变量
func (w *WebhookService) parseHeadersWithVariables(headersStr string, data *WebhookData) map[string]string {
	headers := make(map[string]string)

	if headersStr == "" {
		return headers
	}

	// 解析每一行的header
	lines := strings.Split(headersStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 分割header名和值
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		headerName := strings.TrimSpace(parts[0])
		headerValue := strings.TrimSpace(parts[1])

		// 对header值进行变量替换
		if headerName != "" && headerValue != "" {
			headerValue = w.replaceVariables(headerValue, data)
			headers[headerName] = headerValue
		}
	}

	return headers
}

// handleChineseInURL 专门处理包含中文的URL请求
func (w *WebhookService) handleChineseInURL(urlStr string, data *WebhookData) error {
	// 解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("URL格式不正确: %w", err)
	}

	// 如果没有查询参数，直接使用原始URL
	if parsedURL.RawQuery == "" {
		return nil
	}

	// 获取查询参数content的值
	values := parsedURL.Query()
	contentValue := values.Get("content")

	// 如果没有content参数，直接返回
	if contentValue == "" {
		return nil
	}

	// 特殊处理content参数
	log.Printf("检测到content参数: %s，尝试直接使用resty的QueryParam", contentValue)

	// 创建resty客户端
	client := resty.New().
		SetTimeout(10 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(500 * time.Millisecond)

	// 创建请求
	baseURL := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path
	log.Printf("基础URL: %s", baseURL)

	// 发送请求
	resp, err := client.R().
		SetQueryParam("content", contentValue).
		Get(baseURL)

	if err != nil {
		return fmt.Errorf("发送webhook失败: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("webhook服务返回错误状态码: %d, 响应体: %s", resp.StatusCode(), resp.String())
	}

	log.Printf("webhook请求成功，状态码: %d", resp.StatusCode())
	return nil
}

// sanitizeURL 清理URL中的特殊字符
func (w *WebhookService) sanitizeURL(urlStr string) string {
	// 检测URL中是否含有content参数
	parts := strings.SplitN(urlStr, "?", 2)
	if len(parts) != 2 {
		return urlStr // 没有查询参数，直接返回
	}

	baseURL := parts[0]
	queryStr := parts[1]

	// 解析查询参数
	values, err := url.ParseQuery(queryStr)
	if err != nil {
		// 如果解析失败，尝试手动处理
		log.Printf("解析查询参数失败: %s, 尝试手动处理", err)

		// 分割多个参数
		params := strings.Split(queryStr, "&")
		cleanParams := []string{}

		for _, param := range params {
			// 分割参数名和值
			kv := strings.SplitN(param, "=", 2)
			if len(kv) != 2 {
				cleanParams = append(cleanParams, param) // 保持不变
				continue
			}

			key := kv[0]
			value := kv[1]

			// 对参数值进行清理和编码
			cleanValue := strings.Map(func(r rune) rune {
				// 过滤掉控制字符
				if r < 32 || r == 127 {
					return -1 // 删除字符
				}
				return r
			}, value)

			// 对所有参数值进行URL编码
			encodedValue := url.QueryEscape(cleanValue)
			cleanParams = append(cleanParams, key+"="+encodedValue)
		}

		// 重新组合URL
		return baseURL + "?" + strings.Join(cleanParams, "&")
	}

	// 如果解析成功，确保每个参数值都被正确编码
	encodedQuery := values.Encode()
	return baseURL + "?" + encodedQuery
}
