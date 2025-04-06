package repository

import (
	"backup-go/config"
	"gorm.io/gorm"
)

// GetDB 获取GORM数据库连接
func GetDB() *gorm.DB {
	// 直接使用配置模块的数据库连接
	return config.GetDB()
}

// 为兼容原有代码，提供获取*sql.DB的方法
func GetSqlDB() (*gorm.DB, error) {
	return GetDB(), nil
}
