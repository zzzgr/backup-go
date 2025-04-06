package controller

import (
	"backup-go/entity"
	"backup-go/model"
	"backup-go/repository"
	"backup-go/service/scheduler"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// TaskController 任务控制器
type TaskController struct {
	taskRepo   *repository.BackupTaskRepository
	recordRepo *repository.BackupRecordRepository
	scheduler  *scheduler.BackupScheduler
}

// NewTaskController 创建任务控制器
func NewTaskController() *TaskController {
	return &TaskController{
		taskRepo:   repository.NewBackupTaskRepository(),
		recordRepo: repository.NewBackupRecordRepository(),
		scheduler:  scheduler.GetScheduler(),
	}
}

// CreateTask 创建任务
func (c *TaskController) CreateTask(w http.ResponseWriter, r *http.Request) {
	var task entity.BackupTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		c.writeJSON(w, model.Error(400, "Invalid request: "+err.Error()))
		return
	}

	if err := c.taskRepo.Create(&task); err != nil {
		c.writeJSON(w, model.Error(500, "Failed to create task: "+err.Error()))
		return
	}

	// 添加到调度器
	if task.Enabled {
		if err := c.scheduler.AddTask(&task); err != nil {
			c.writeJSON(w, model.Error(500, "Failed to schedule task: "+err.Error()))
			return
		}
	}

	c.writeJSON(w, model.Success(task))
}

// UpdateTask 更新任务
func (c *TaskController) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "Invalid task ID"))
		return
	}

	// 获取任务
	task, err := c.taskRepo.FindByID(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to find task: "+err.Error()))
		return
	}
	if task == nil {
		c.writeJSON(w, model.Error(404, "Task not found"))
		return
	}

	// 解析更新数据
	var updatedTask entity.BackupTask
	if err := json.NewDecoder(r.Body).Decode(&updatedTask); err != nil {
		c.writeJSON(w, model.Error(400, "Invalid request: "+err.Error()))
		return
	}

	// 更新数据
	updatedTask.ID = id
	if err := c.taskRepo.Update(&updatedTask); err != nil {
		c.writeJSON(w, model.Error(500, "Failed to update task: "+err.Error()))
		return
	}

	// 更新调度
	c.scheduler.RemoveTask(id)
	if updatedTask.Enabled {
		if err := c.scheduler.AddTask(&updatedTask); err != nil {
			c.writeJSON(w, model.Error(500, "Failed to schedule task: "+err.Error()))
			return
		}
	}

	c.writeJSON(w, model.Success(updatedTask))
}

// DeleteTask 删除任务
func (c *TaskController) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "Invalid task ID"))
		return
	}

	// 从调度器中移除
	c.scheduler.RemoveTask(id)

	// 开始事务
	tx := repository.GetDB().Begin()
	if tx.Error != nil {
		c.writeJSON(w, model.Error(500, "开始事务失败: "+tx.Error.Error()))
		return
	}

	// 删除关联的所有备份记录
	if err := tx.Where("task_id = ?", id).Delete(&entity.BackupRecord{}).Error; err != nil {
		tx.Rollback()
		c.writeJSON(w, model.Error(500, "删除任务关联的备份记录失败: "+err.Error()))
		return
	}

	// 删除任务
	if err := tx.Delete(&entity.BackupTask{}, id).Error; err != nil {
		tx.Rollback()
		c.writeJSON(w, model.Error(500, "删除任务失败: "+err.Error()))
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.writeJSON(w, model.Error(500, "提交事务失败: "+err.Error()))
		return
	}

	c.writeJSON(w, model.Success(nil))
}

// GetTask 获取任务
func (c *TaskController) GetTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "Invalid task ID"))
		return
	}

	task, err := c.taskRepo.FindByID(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to find task: "+err.Error()))
		return
	}
	if task == nil {
		c.writeJSON(w, model.Error(404, "Task not found"))
		return
	}

	c.writeJSON(w, model.Success(task))
}

// GetAllTasks 获取所有任务
func (c *TaskController) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	// 获取分页参数
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// 检查是否请求不分页的数据
	noPagination := r.URL.Query().Get("noPagination") == "true"

	var tasks []*entity.BackupTask
	var total int64
	var err2 error

	if noPagination {
		// 不分页，获取所有任务
		tasks, err2 = c.taskRepo.FindAll()
		if err2 != nil {
			c.writeJSON(w, model.Error(500, "Failed to find tasks: "+err2.Error()))
			return
		}

		// 给每个任务添加下一次执行时间
		c.enrichTasksWithNextExecutionTime(tasks)

		c.writeJSON(w, model.Success(tasks))
	} else {
		// 获取分页数据
		tasks, err2 = c.taskRepo.FindAllPaginated(page, pageSize)
		if err2 != nil {
			c.writeJSON(w, model.Error(500, "Failed to find tasks: "+err2.Error()))
			return
		}

		// 获取总任务数
		total, err2 = c.taskRepo.CountAll()
		if err2 != nil {
			c.writeJSON(w, model.Error(500, "Failed to count tasks: "+err2.Error()))
			return
		}

		// 给每个任务添加下一次执行时间
		c.enrichTasksWithNextExecutionTime(tasks)

		// 返回分页结果
		result := map[string]interface{}{
			"tasks":    tasks,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		}

		c.writeJSON(w, model.Success(result))
	}
}

// enrichTasksWithNextExecutionTime 给任务添加下一次执行时间
func (c *TaskController) enrichTasksWithNextExecutionTime(tasks []*entity.BackupTask) {
	// 为每个任务添加下一次执行时间
	for _, task := range tasks {
		// 任务已启用时才计算下一次执行时间
		if task.Enabled {
			nextTime := c.scheduler.GetNextExecutionTime(task.ID)
			if nextTime != nil {
				// 添加到任务的额外数据中
				if task.ExtraData == nil {
					task.ExtraData = make(map[string]interface{})
				}
				task.ExtraData["nextExecutionTime"] = nextTime.Format("2006-01-02 15:04:05")
			}
		} else {
			// 如果任务未启用，设置为已禁用
			if task.ExtraData == nil {
				task.ExtraData = make(map[string]interface{})
			}
			task.ExtraData["nextExecutionTime"] = "已禁用"
		}
	}
}

// ExecuteTask 立即执行任务
func (c *TaskController) ExecuteTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "Invalid task ID"))
		return
	}

	// 检查任务是否存在
	task, err := c.taskRepo.FindByID(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "Failed to find task: "+err.Error()))
		return
	}
	if task == nil {
		c.writeJSON(w, model.Error(404, "Task not found"))
		return
	}

	// 立即执行
	if err := c.scheduler.ExecuteTaskNow(id); err != nil {
		c.writeJSON(w, model.Error(500, "Failed to execute task: "+err.Error()))
		return
	}

	c.writeJSON(w, model.Success(nil))
}

// UpdateTaskEnabled 更新任务启用状态
func (c *TaskController) UpdateTaskEnabled(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "无效的任务ID"))
		return
	}

	// 获取请求中的enabled状态
	enabledStr := r.URL.Query().Get("enabled")
	if enabledStr == "" {
		c.writeJSON(w, model.Error(400, "缺少启用状态参数"))
		return
	}

	enabled := enabledStr == "true"

	// 获取任务
	task, err := c.taskRepo.FindByID(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "查找任务失败: "+err.Error()))
		return
	}
	if task == nil {
		c.writeJSON(w, model.Error(404, "任务不存在"))
		return
	}

	// 如果状态没有变化，直接返回成功
	if task.Enabled == enabled {
		c.writeJSON(w, model.Success(task))
		return
	}

	// 更新任务启用状态
	task.Enabled = enabled
	task.UpdatedAt = time.Now()

	if err := c.taskRepo.Update(task); err != nil {
		c.writeJSON(w, model.Error(500, "更新任务状态失败: "+err.Error()))
		return
	}

	// 处理调度器中的任务
	if enabled {
		// 启用任务，添加到调度器
		if err := c.scheduler.AddTask(task); err != nil {
			c.writeJSON(w, model.Error(500, "添加任务到调度器失败: "+err.Error()))
			return
		}
		log.Printf("任务 %d 已启用并添加到调度器", id)
	} else {
		// 禁用任务，从调度器移除
		c.scheduler.RemoveTask(id)
		log.Printf("任务 %d 已禁用并从调度器移除", id)
	}

	c.writeJSON(w, model.Success(task))
}

// GetTaskNextExecutionTime 获取任务的下一次执行时间
func (c *TaskController) GetTaskNextExecutionTime(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		c.writeJSON(w, model.Error(400, "无效的任务ID"))
		return
	}

	// 获取任务
	task, err := c.taskRepo.FindByID(id)
	if err != nil {
		c.writeJSON(w, model.Error(500, "查找任务失败: "+err.Error()))
		return
	}
	if task == nil {
		c.writeJSON(w, model.Error(404, "任务不存在"))
		return
	}

	// 获取下一次执行时间
	nextTime := c.scheduler.GetNextExecutionTime(id)
	if nextTime == nil {
		c.writeJSON(w, model.Success(nil))
		return
	}

	c.writeJSON(w, model.Success(nextTime.Format("2006-01-02 15:04:05")))
}

// 写入JSON响应
func (c *TaskController) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
