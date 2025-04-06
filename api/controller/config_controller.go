package controller

import (
	"backup-go/entity"
	"backup-go/model"
	"backup-go/service/config"
	"encoding/json"
	"net/http"
	"strconv"
)

// ConfigController 配置控制器
type ConfigController struct {
	configService  *config.ConfigService
	webhookService *config.WebhookService
}

// NewConfigController 创建控制器实例
func NewConfigController() *ConfigController {
	return &ConfigController{
		configService:  config.NewConfigService(),
		webhookService: config.NewWebhookService(),
	}
}

// GetConfigList 获取配置列表
// @Summary 获取配置列表
// @Description 获取所有配置项
// @Tags 配置管理
// @Produce json
// @Success 200 {object} model.Response
// @Router /api/configs [get]
func (c *ConfigController) GetConfigList(w http.ResponseWriter, r *http.Request) {
	configs, err := c.configService.GetAllConfigs()
	if err != nil {
		WriteJSONResponse(w, model.FailResponse(err.Error()))
		return
	}
	WriteJSONResponse(w, model.SuccessResponse(configs))
}

// GetConfig 获取单个配置
// @Summary 获取单个配置
// @Description 根据ID获取配置详情
// @Tags 配置管理
// @Produce json
// @Param id query int true "配置ID"
// @Success 200 {object} model.Response
// @Router /api/configs/get [get]
func (c *ConfigController) GetConfig(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteJSONResponse(w, model.FailResponse("无效的ID"))
		return
	}

	config, err := c.configService.GetConfig(uint(id))
	if err != nil {
		WriteJSONResponse(w, model.FailResponse(err.Error()))
		return
	}

	WriteJSONResponse(w, model.SuccessResponse(config))
}

// GetConfigByKey 根据键获取配置
// @Summary 根据键获取配置
// @Description 根据键名获取配置详情
// @Tags 配置管理
// @Produce json
// @Param key query string true "配置键"
// @Success 200 {object} model.Response
// @Router /api/configs/getByKey [get]
func (c *ConfigController) GetConfigByKey(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		WriteJSONResponse(w, model.FailResponse("配置键不能为空"))
		return
	}

	config, err := c.configService.GetConfigByKey(key)
	if err != nil {
		WriteJSONResponse(w, model.FailResponse(err.Error()))
		return
	}

	WriteJSONResponse(w, model.SuccessResponse(config))
}

// CreateConfig 创建配置
// @Summary 创建配置
// @Description 创建新的配置项
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param config body entity.SystemConfig true "配置信息"
// @Success 200 {object} model.Response
// @Router /api/configs [post]
func (c *ConfigController) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var config entity.SystemConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		WriteJSONResponse(w, model.FailResponse("无效的请求数据: "+err.Error()))
		return
	}

	// 验证必填字段
	if config.ConfigKey == "" {
		WriteJSONResponse(w, model.FailResponse("配置键不能为空"))
		return
	}

	// 检查键是否已存在
	existingConfig, _ := c.configService.GetConfigByKey(config.ConfigKey)
	if existingConfig != nil {
		WriteJSONResponse(w, model.FailResponse("配置键已存在"))
		return
	}

	// 创建配置
	if err := c.configService.CreateConfig(&config); err != nil {
		WriteJSONResponse(w, model.FailResponse("创建配置失败: "+err.Error()))
		return
	}

	WriteJSONResponse(w, model.SuccessResponse(config))
}

// UpdateConfig 更新配置
// @Summary 更新配置
// @Description 更新现有配置项
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param id query int true "配置ID"
// @Param config body entity.SystemConfig true "配置信息"
// @Success 200 {object} model.Response
// @Router /api/configs/update [post]
func (c *ConfigController) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteJSONResponse(w, model.FailResponse("无效的ID"))
		return
	}

	// 获取现有配置
	existingConfig, err := c.configService.GetConfig(uint(id))
	if err != nil {
		WriteJSONResponse(w, model.FailResponse("配置不存在"))
		return
	}

	// 解析请求数据
	var updateData entity.SystemConfig
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		WriteJSONResponse(w, model.FailResponse("无效的请求数据: "+err.Error()))
		return
	}

	// 验证必填字段
	if updateData.ConfigKey == "" {
		WriteJSONResponse(w, model.FailResponse("配置键不能为空"))
		return
	}

	// 如果修改了键名，检查新键名是否已存在
	if updateData.ConfigKey != existingConfig.ConfigKey {
		checkConfig, _ := c.configService.GetConfigByKey(updateData.ConfigKey)
		if checkConfig != nil {
			WriteJSONResponse(w, model.FailResponse("配置键已存在"))
			return
		}
	}

	// 只更新需要的字段，保留其他原有信息
	existingConfig.ConfigKey = updateData.ConfigKey
	existingConfig.ConfigValue = updateData.ConfigValue
	existingConfig.Description = updateData.Description

	// 更新配置
	if err := c.configService.UpdateConfig(existingConfig); err != nil {
		WriteJSONResponse(w, model.FailResponse("更新配置失败: "+err.Error()))
		return
	}

	WriteJSONResponse(w, model.SuccessResponse(existingConfig))
}

// DeleteConfig 删除配置
// @Summary 删除配置
// @Description 删除配置项
// @Tags 配置管理
// @Produce json
// @Param id query int true "配置ID"
// @Success 200 {object} model.Response
// @Router /api/configs/delete [post]
func (c *ConfigController) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		WriteJSONResponse(w, model.FailResponse("无效的ID"))
		return
	}

	// 检查是否存在
	_, err = c.configService.GetConfig(uint(id))
	if err != nil {
		WriteJSONResponse(w, model.FailResponse("配置不存在"))
		return
	}

	// 删除配置
	if err := c.configService.DeleteConfig(uint(id)); err != nil {
		WriteJSONResponse(w, model.FailResponse("删除配置失败: "+err.Error()))
		return
	}

	WriteJSONResponse(w, model.SuccessResponse(nil))
}

// TestWebhook 测试Webhook
// @Summary 测试Webhook
// @Description 测试Webhook配置是否正确
// @Tags 配置管理
// @Produce json
// @Success 200 {object} model.Response
// @Router /api/configs/testWebhook [post]
func (c *ConfigController) TestWebhook(w http.ResponseWriter, r *http.Request) {
	// 解析请求体中的配置
	var tempConfig struct {
		URL     string `json:"url"`
		Headers string `json:"headers"`
		Body    string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&tempConfig); err != nil {
		WriteJSONResponse(w, model.FailResponse("解析请求参数失败: "+err.Error()))
		return
	}

	// 验证URL是否存在
	if tempConfig.URL == "" {
		WriteJSONResponse(w, model.FailResponse("URL不能为空"))
		return
	}

	// 使用表单配置进行测试
	err := c.webhookService.TestWebhookWithConfig(tempConfig.URL, tempConfig.Headers, tempConfig.Body)
	if err != nil {
		WriteJSONResponse(w, model.FailResponse("测试Webhook失败: "+err.Error()))
		return
	}

	WriteJSONResponse(w, model.SuccessResponse(nil))
}

// GetSiteInfo 获取站点信息
// @Summary 获取站点信息
// @Description 获取站点名称和版本号信息，此接口可匿名访问
// @Tags 系统信息
// @Produce json
// @Success 200 {object} model.Response
// @Router /api/info [get]
func (c *ConfigController) GetSiteInfo(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头，允许跨域访问
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 获取站点名称
	siteName, err := c.configService.GetConfigValueByKey("system.siteName")
	if err != nil || siteName == "" {
		siteName = "备份系统" // 默认站点名称
	}

	// 硬编码版本号
	version := "v1.0.0" // 系统版本号，硬编码

	// 构建响应数据
	data := map[string]string{
		"siteName": siteName,
		"version":  version,
	}

	WriteJSONResponse(w, model.SuccessResponse(data))
}

// 通用JSON响应写入函数
