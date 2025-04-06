package backup

import (
	"backup-go/entity"
	"backup-go/repository"
	"fmt"
	"log"
	"time"
)

// 处理结果统计
type processStats struct {
	runningCount   int // 处理的运行中记录数
	pendingCount   int // 处理的等待中记录数
	processFailed  int // 处理失败的记录数
	processSuccess int // 处理成功的记录数
}

// InitBackupRecords 处理系统启动时处于异常状态的备份记录
// 包括"运行中"和"等待中"但超过一定时间的记录
func InitBackupRecords() error {
	stats := &processStats{}

	// 处理运行中的记录
	if err := initRunningRecords(stats); err != nil {
		return err
	}

	// 处理长时间等待中的记录
	if err := initPendingRecords(stats); err != nil {
		return err
	}

	// 输出统计信息
	log.Printf("备份记录处理统计: 总共处理 %d 条记录 (运行中: %d, 等待中: %d), 成功: %d, 失败: %d",
		stats.runningCount+stats.pendingCount,
		stats.runningCount,
		stats.pendingCount,
		stats.processSuccess,
		stats.processFailed)

	return nil
}

// initRunningRecords 处理系统启动时处于"运行中"状态的备份记录
// 这些记录可能是由于系统非正常退出导致的，需要将它们标记为失败
func initRunningRecords(stats *processStats) error {
	recordRepo := repository.NewBackupRecordRepository()

	// 查找所有处于"运行中"状态的记录
	runningRecords, err := recordRepo.FindByStatus(entity.StatusRunning)
	if err != nil {
		return fmt.Errorf("查询运行中的备份记录失败: %w", err)
	}

	stats.runningCount = len(runningRecords)
	log.Printf("找到 %d 条处于运行中状态的备份记录，将标记为失败", stats.runningCount)

	// 处理每一条记录
	for _, record := range runningRecords {
		// 设置记录状态为失败
		record.Status = entity.StatusFailed
		record.EndTime = time.Now()
		record.ErrorMessage = "系统重启时，该任务处于运行中状态，已被自动标记为失败"

		// 更新记录
		if err := recordRepo.Update(record); err != nil {
			log.Printf("更新备份记录 ID=%d 失败: %v", record.ID, err)
			stats.processFailed++
			continue
		}

		log.Printf("已将备份记录 ID=%d 从'运行中'状态标记为'失败'", record.ID)
		stats.processSuccess++
	}

	return nil
}

// initPendingRecords 处理系统启动时处于"等待中"状态且超过1小时的备份记录
// 这些记录可能是由于系统异常导致无法启动的任务
func initPendingRecords(stats *processStats) error {
	recordRepo := repository.NewBackupRecordRepository()

	// 查找所有处于"等待中"状态的记录
	pendingRecords, err := recordRepo.FindByStatus(entity.StatusPending)
	if err != nil {
		return fmt.Errorf("查询等待中的备份记录失败: %w", err)
	}

	// 当前时间
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	// 统计需要处理的记录数
	for _, record := range pendingRecords {
		if record.StartTime.Before(oneHourAgo) {
			stats.pendingCount++
		}
	}

	log.Printf("找到 %d 条长时间（超过1小时）处于等待中状态的备份记录，将标记为失败", stats.pendingCount)

	// 处理每一条记录
	for _, record := range pendingRecords {
		// 只处理超过1小时的待处理记录
		if record.StartTime.Before(oneHourAgo) {
			// 设置记录状态为失败
			record.Status = entity.StatusFailed
			record.EndTime = now
			record.ErrorMessage = "系统重启时，该任务长时间处于等待中状态，已被自动标记为失败"

			// 更新记录
			if err := recordRepo.Update(record); err != nil {
				log.Printf("更新备份记录 ID=%d 失败: %v", record.ID, err)
				stats.processFailed++
				continue
			}

			log.Printf("已将备份记录 ID=%d 从'等待中'状态标记为'失败'", record.ID)
			stats.processSuccess++
		}
	}

	return nil
}
