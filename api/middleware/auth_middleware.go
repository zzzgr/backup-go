package middleware

import (
	"backup-go/api/controller"
	"backup-go/model"
	"backup-go/service/config"
	"encoding/json"
	"net/http"
	"strings"
)

// AuthMiddleware 身份验证中间件
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 跳过登录接口的身份验证
		if r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/check" || r.URL.Path == "/api/records/download" {
			next.ServeHTTP(w, r)
			return
		}

		// 检查是否启用密码保护
		configService := config.NewConfigService()
		passwordConfig, err := configService.GetConfigByKey("system.password")

		// 如果没有设置密码或配置不存在，不需要验证
		if err != nil || passwordConfig.ConfigValue == "" {
			next.ServeHTTP(w, r)
			return
		}

		// 获取并验证token
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			// 没有提供token，返回未授权错误
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(model.ErrorWithCode(401, "未授权访问"))
			return
		}

		// 验证token格式
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(model.ErrorWithCode(401, "无效的凭证格式"))
			return
		}

		// 提取token值（去除Bearer前缀）
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// 验证token是否有效
		if !controller.IsValidToken(token) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(model.ErrorWithCode(401, "无效的凭证或已过期"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
