package controller

import (
	"backup-go/model"
	"backup-go/service/config"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 用于存储活跃token的内存映射表
var (
	activeTokens = make(map[string]time.Time) // token -> 过期时间
	tokenMutex   = &sync.RWMutex{}
)

// 令牌过期时间（24小时）
const tokenExpiration = 24 * time.Hour

// AuthController 身份验证控制器
type AuthController struct {
	configService *config.ConfigService
}

// NewAuthController 创建控制器实例
func NewAuthController() *AuthController {
	return &AuthController{
		configService: config.NewConfigService(),
	}
}

// Login 用户登录
// @Summary 用户登录
// @Description 根据密码登录系统
// @Tags 身份验证
// @Accept json
// @Produce json
// @Param data body loginRequest true "登录信息"
// @Success 200 {object} model.Response
// @Router /api/auth/login [post]
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSONResponse(w, model.FailResponse("无效的请求数据"))
		return
	}

	// 获取系统密码配置
	systemPasswordConfig, err := c.configService.GetConfigByKey("system.password")
	if err != nil {
		// 未设置密码，默认允许登录
		token := generateToken()
		WriteJSONResponse(w, model.SuccessResponse(map[string]string{
			"token": token,
		}))
		return
	}

	// 验证密码
	if req.Password != systemPasswordConfig.ConfigValue {
		WriteJSONResponse(w, model.FailResponse("密码错误"))
		return
	}

	// 生成token
	token := generateToken()

	// 返回成功响应
	WriteJSONResponse(w, model.SuccessResponse(map[string]string{
		"token": token,
	}))
}

// CheckAuth 验证用户身份
// @Summary 验证用户身份
// @Description 验证用户是否已登录
// @Tags 身份验证
// @Produce json
// @Success 200 {object} model.Response
// @Router /api/auth/check [get]
func (c *AuthController) CheckAuth(w http.ResponseWriter, r *http.Request) {
	// 获取系统密码配置
	systemPasswordConfig, err := c.configService.GetConfigByKey("system.password")
	if err != nil {
		// 未设置密码，不需要验证
		WriteJSONResponse(w, model.SuccessResponse(nil))
		return
	}

	// 检查密码是否为空
	if systemPasswordConfig.ConfigValue == "" {
		// 密码为空，不需要验证
		WriteJSONResponse(w, model.SuccessResponse(nil))
		return
	}

	// 获取token
	token := r.Header.Get("Authorization")
	if token == "" {
		WriteJSONResponse(w, model.ErrorWithCode(401, "未授权"))
		return
	}

	// 验证token格式
	if !strings.HasPrefix(token, "Bearer ") {
		WriteJSONResponse(w, model.ErrorWithCode(401, "无效的凭证格式"))
		return
	}

	// 提取token值（去除Bearer前缀）
	tokenValue := strings.TrimPrefix(token, "Bearer ")

	// 验证token是否有效
	if !IsValidToken(tokenValue) {
		WriteJSONResponse(w, model.ErrorWithCode(401, "无效的凭证或已过期"))
		return
	}

	// 验证通过
	WriteJSONResponse(w, model.SuccessResponse(nil))
}

// loginRequest 登录请求结构
type loginRequest struct {
	Password string `json:"password"`
}

// generateToken 生成UUID作为token
func generateToken() string {
	// 生成UUID作为token
	token := uuid.New().String()

	// 保存到活跃token映射表中
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	// 设置过期时间
	activeTokens[token] = time.Now().Add(tokenExpiration)

	// 清理过期token
	cleanExpiredTokens()

	return token
}

// isValidToken 检查token是否有效
func IsValidToken(token string) bool {
	tokenMutex.RLock()
	defer tokenMutex.RUnlock()

	expireTime, exists := activeTokens[token]
	return exists && time.Now().Before(expireTime)
}

// cleanExpiredTokens 清理过期的token
func cleanExpiredTokens() {
	now := time.Now()
	for token, expireTime := range activeTokens {
		if now.After(expireTime) {
			delete(activeTokens, token)
		}
	}
}

// WriteJSONResponse 写入JSON响应
func WriteJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
