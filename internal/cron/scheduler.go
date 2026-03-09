package cron

import (
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

// BackupFunc 是定时任务触发时要执行的备份函数签名。
type BackupFunc func()

// Scheduler 管理定时备份任务。
type Scheduler struct {
	mu       sync.RWMutex
	c        *cron.Cron
	entryID  cron.EntryID
	running  bool
	expr     string
	backupFn BackupFunc
}

// NewScheduler 创建新的调度器实例。
func NewScheduler(fn BackupFunc) *Scheduler {
	return &Scheduler{
		c:        cron.New(cron.WithSeconds()),
		backupFn: fn,
	}
}

// Start 使用给定的 Cron 表达式启动调度器。
// 如果已有调度在运行，会无缝替换为新表达式。
func (s *Scheduler) Start(expression string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 先删除旧任务，避免新旧任务同时存在的短暂窗口
	if s.entryID != 0 {
		s.c.Remove(s.entryID)
		s.entryID = 0
	}

	entryID, err := s.c.AddFunc(expression, func() {
		log.Printf("[cron] triggered backup job: %s", expression)
		if s.backupFn != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[cron] backup callback panic recovered: %v", r)
					}
				}()
				s.backupFn()
			}()
		}
	})
	if err != nil {
		return err
	}

	s.entryID = entryID
	s.expr = expression
	if !s.running {
		s.c.Start()
		s.running = true
	}

	log.Printf("[cron] scheduler started with expression: %s", expression)
	return nil
}

// Stop 停止调度器。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.entryID != 0 {
		s.c.Remove(s.entryID)
		s.entryID = 0
	}

	if s.running {
		s.c.Stop()
		s.running = false
		log.Println("[cron] scheduler stopped")
	}
}

// IsRunning 返回调度器是否在运行。
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// NextRun 返回下次执行时间的字符串表示。
func (s *Scheduler) NextRun() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return ""
	}

	entries := s.c.Entries()
	for _, e := range entries {
		if e.ID == s.entryID {
			return e.Next.Format("2006-01-02 15:04:05")
		}
	}
	return ""
}

// Expression 返回当前的 Cron 表达式。
func (s *Scheduler) Expression() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.expr
}
