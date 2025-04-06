package scheduler

import (
	"backup-go/entity"
	"backup-go/repository"
	"backup-go/service/backup"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// BackupScheduler 备份调度器
type BackupScheduler struct {
	cron       *cron.Cron
	taskRepo   *repository.BackupTaskRepository
	recordRepo *repository.BackupRecordRepository
	jobs       map[int64]cron.EntryID
	mutex      sync.Mutex
	running    bool
}

var (
	scheduler *BackupScheduler
	once      sync.Once
)

// GetScheduler 获取单例的调度器
func GetScheduler() *BackupScheduler {
	once.Do(func() {
		scheduler = &BackupScheduler{
			cron:       cron.New(cron.WithSeconds()),
			taskRepo:   repository.NewBackupTaskRepository(),
			recordRepo: repository.NewBackupRecordRepository(),
			jobs:       make(map[int64]cron.EntryID),
		}
	})
	return scheduler
}

// Start 启动调度器
func (s *BackupScheduler) Start() {
	s.mutex.Lock()
	// 检查是否已经运行
	if s.running {
		s.mutex.Unlock()
		return
	}
	s.mutex.Unlock()

	// 加载任务 - 在锁外执行
	if err := s.loadTasks(); err != nil {
		log.Printf("加载任务失败: %v", err)
	}

	s.mutex.Lock()
	// 启动Cron
	s.cron.Start()
	s.running = true
	s.mutex.Unlock()

	log.Println("调度器启动成功")
}

// Stop 停止调度器
func (s *BackupScheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	// 停止并等待所有任务完成
	ctx := s.cron.Stop()
	<-ctx.Done()

	s.running = false
	log.Println("调度器已停止")
}

// Reload 重新加载所有任务
func (s *BackupScheduler) Reload() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 移除所有任务
	for taskID, entryID := range s.jobs {
		s.cron.Remove(entryID)
		delete(s.jobs, taskID)
	}

	// 加载任务
	return s.loadTasks()
}

// AddTask 添加任务
func (s *BackupScheduler) AddTask(task *entity.BackupTask) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 如果任务已存在，先移除
	if entryID, exists := s.jobs[task.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, task.ID)
	}

	// 如果任务未启用，则不添加
	if !task.Enabled {
		return nil
	}

	// 添加任务
	entryID, err := s.cron.AddFunc(task.Schedule, func() {
		s.executeTask(task.ID)
	})

	if err != nil {
		return fmt.Errorf("failed to add task to scheduler: %w", err)
	}

	s.jobs[task.ID] = entryID
	return nil
}

// RemoveTask 移除任务
func (s *BackupScheduler) RemoveTask(taskID int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if entryID, exists := s.jobs[taskID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, taskID)
	}
}

// IsTaskScheduled 检查任务是否已调度
func (s *BackupScheduler) IsTaskScheduled(taskID int64) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, exists := s.jobs[taskID]
	return exists
}

// GetScheduledTasks 获取所有调度任务ID
func (s *BackupScheduler) GetScheduledTasks() []int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var taskIDs []int64
	for taskID := range s.jobs {
		taskIDs = append(taskIDs, taskID)
	}
	return taskIDs
}

// ExecuteTaskNow 立即执行任务
func (s *BackupScheduler) ExecuteTaskNow(taskID int64) error {
	go s.executeTask(taskID)
	return nil
}

// 加载所有启用的任务
func (s *BackupScheduler) loadTasks() error {
	tasks, err := s.taskRepo.GetEnabledTasks()
	if err != nil {
		return fmt.Errorf("failed to get enabled tasks: %w", err)
	}

	for _, task := range tasks {
		// 直接在loadTasks内部处理任务添加，避免调用AddTask方法
		// 检查任务是否已存在
		s.mutex.Lock()
		if entryID, exists := s.jobs[task.ID]; exists {
			s.cron.Remove(entryID)
			delete(s.jobs, task.ID)
		}

		// 如果任务未启用，则不添加
		if !task.Enabled {
			s.mutex.Unlock()
			continue
		}

		// 添加任务
		entryID, err := s.cron.AddFunc(task.Schedule, func(taskID int64) func() {
			return func() {
				s.executeTask(taskID)
			}
		}(task.ID))

		if err != nil {
			log.Printf("添加任务 %d 到调度器失败: %v", task.ID, err)
			s.mutex.Unlock()
			continue
		}

		s.jobs[task.ID] = entryID
		s.mutex.Unlock()
	}

	return nil
}

// 执行任务
func (s *BackupScheduler) executeTask(taskID int64) {
	log.Printf("开始执行任务 %d", taskID)

	// 获取任务
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		log.Printf("获取任务 %d 失败: %v", taskID, err)
		return
	}

	if task == nil {
		log.Printf("任务 %d 不存在", taskID)
		return
	}

	// 创建备份服务
	backupService, err := backup.NewBackupService(task.Type)
	if err != nil {
		log.Printf("为任务 %d 创建备份服务失败: %v", taskID, err)
		return
	}

	// 执行备份
	record, err := backupService.Execute(task)
	if err != nil {
		log.Printf("执行任务 %d 的备份失败: %v", taskID, err)
		// 记录已经在备份服务中更新过了
		return
	}

	log.Printf("任务 %d 执行成功，备份记录ID: %d", taskID, record.ID)
}

// GetNextExecutionTime 获取任务的下一次执行时间
func (s *BackupScheduler) GetNextExecutionTime(taskID int64) *time.Time {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if entryID, exists := s.jobs[taskID]; exists {
		entry := s.cron.Entry(entryID)
		next := entry.Next
		return &next
	}
	return nil
}
