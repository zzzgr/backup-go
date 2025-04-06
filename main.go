package main

import (
	"backup-go/api/router"
	"backup-go/config"
	backupService "backup-go/service/backup"
	"backup-go/service/cleanup"
	configService "backup-go/service/config"
	"backup-go/service/scheduler"
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 连接数据库
	if err := config.InitDB(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 执行数据库迁移
	if err := config.MigrateDB(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 初始化系统默认配置
	cs := configService.NewConfigService()
	if err := cs.InitDefaultConfigs(); err != nil {
		log.Printf("初始化默认配置失败: %v", err)
	}

	// 处理异常状态的备份记录
	if err := backupService.InitBackupRecords(); err != nil {
		log.Printf("处理异常备份记录失败: %v", err)
	}

	// 启动调度器
	backupScheduler := scheduler.GetScheduler()
	backupScheduler.Start()

	// 启动清理服务
	cleanupSvc := cleanup.GetCleanupService()
	cleanupSvc.Start()

	// 设置路由
	r := router.SetupRouter()

	// 启动HTTP服务
	serverAddr := fmt.Sprintf(":%d", config.Get().Server.Port)
	log.Printf("HTTP服务启动%s\n", serverAddr)
	if err := http.ListenAndServe(serverAddr, r); err != nil {
		// 停止调度器
		backupScheduler.Stop()
		// 停止清理服务
		cleanupSvc.Stop()
		log.Fatalf("服务启动失败: %v", err)
	}
}
