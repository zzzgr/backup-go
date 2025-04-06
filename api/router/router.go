package router

import (
	"backup-go/api/controller"
	"backup-go/api/middleware"
	"net/http"
)

// SetupRouter 设置路由
func SetupRouter() http.Handler {
	taskController := controller.NewTaskController()
	recordController := controller.NewRecordController()
	configController := controller.NewConfigController()
	authController := controller.NewAuthController()
	cleanupController := controller.NewCleanupController()

	// 创建路由复用器
	mux := http.NewServeMux()

	// 静态文件服务
	mux.Handle("/", http.FileServer(http.Dir("public")))

	// 添加匿名访问的API路由
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			configController.GetSiteInfo(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// API路由
	apiRoutes := http.NewServeMux()

	// 认证相关路由
	apiRoutes.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authController.Login(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/auth/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authController.CheckAuth(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 任务相关路由
	apiRoutes.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			taskController.GetAllTasks(w, r)
		case http.MethodPost:
			taskController.CreateTask(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/tasks/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			taskController.GetTask(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/tasks/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			taskController.UpdateTask(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/tasks/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			taskController.DeleteTask(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/tasks/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			taskController.ExecuteTask(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/tasks/updateEnabled", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodGet {
			taskController.UpdateTaskEnabled(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/tasks/nextExecutionTime", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			taskController.GetTaskNextExecutionTime(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 记录相关路由
	apiRoutes.HandleFunc("/api/records", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			recordController.GetAllRecords(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/records/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			recordController.GetRecord(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/records/task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			recordController.GetTaskRecords(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/records/download", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			recordController.DownloadBackup(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/records/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			recordController.DeleteRecord(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 配置相关路由
	apiRoutes.HandleFunc("/api/configs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			configController.GetConfigList(w, r)
		case http.MethodPost:
			configController.CreateConfig(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/configs/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			configController.GetConfig(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/configs/getByKey", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			configController.GetConfigByKey(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/configs/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			configController.UpdateConfig(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/configs/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			configController.DeleteConfig(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiRoutes.HandleFunc("/api/configs/testWebhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			configController.TestWebhook(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 清理功能路由
	apiRoutes.HandleFunc("/api/cleanup/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			cleanupController.ExecuteCleanup(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 设置API中间件
	handler := middleware.CorsMiddleware(apiRoutes)
	handler = middleware.LoggingMiddleware(handler)
	handler = middleware.AuthMiddleware(handler)

	// 注册API路由到主路由
	mux.Handle("/api/", handler)

	return mux
}
