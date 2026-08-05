// Package scheduler 定时任务调度：token 刷新、健康检测、保活、收信。
// 间隔可在运行时通过 settings 表覆盖（前端可改），每分钟检查一次是否到点。
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"outlook-manager/internal/config"
	"outlook-manager/internal/model"
	"outlook-manager/internal/service"
)

const (
	keyEnabled   = "sched.enabled"
	keyRefresh   = "sched.refresh_interval"
	keyHealth    = "sched.health_interval"
	keyKeepalive = "sched.keepalive_interval"
	keyMail      = "sched.mail_interval"

	keyLogRetention = "log.retention" // 日志保留期（Go duration；"0" = 永久保留）

	lastRunPrefix = "sched.last_run."
)

// defaultLogRetention 未设置时的默认保留期：30 天。
const defaultLogRetention = 30 * 24 * time.Hour

// Scheduler 定时任务管理器。
type Scheduler struct {
	svc     *service.Service
	cfg     *config.Config
	log     *slog.Logger
	cron    *cron.Cron
	mu      sync.Mutex // 同一时刻仅允许一个批量任务
	stagger time.Duration
}

func New(svc *service.Service, cfg *config.Config, log *slog.Logger) *Scheduler {
	return &Scheduler{
		svc:     svc,
		cfg:     cfg,
		log:     log,
		cron:    cron.New(),
		stagger: 3 * time.Second, // 账号间错峰，降低风控概率
	}
}

// Start 启动调度（每分钟检查一次到点任务）。
func (s *Scheduler) Start() {
	if !s.enabled() {
		s.log.Info("调度器已禁用（sched.enabled=false）")
		return
	}
	_, _ = s.cron.AddFunc("@every 1m", s.tick)
	s.cron.Start()
	s.log.Info("调度器已启动",
		"refresh", s.interval(keyRefresh, s.cfg.Scheduler.RefreshInterval),
		"health", s.interval(keyHealth, s.cfg.Scheduler.HealthInterval),
		"keepalive", s.interval(keyKeepalive, s.cfg.Scheduler.KeepaliveInterval),
		"mail", s.interval(keyMail, s.cfg.Scheduler.MailInterval))
}

// Stop 停止调度。
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

func (s *Scheduler) tick() {
	s.checkAndRun(model.TaskRefresh, keyRefresh, s.cfg.Scheduler.RefreshInterval,
		func(ctx context.Context) service.BatchResult { return s.svc.RefreshAll(ctx, "", s.stagger) })
	s.checkAndRun(model.TaskHealth, keyHealth, s.cfg.Scheduler.HealthInterval,
		func(ctx context.Context) service.BatchResult { return s.svc.CheckAll(ctx, "", s.stagger) })
	s.checkAndRun(model.TaskKeepalive, keyKeepalive, s.cfg.Scheduler.KeepaliveInterval,
		func(ctx context.Context) service.BatchResult { return s.svc.KeepaliveAll(ctx, s.stagger) })
	s.checkAndRun(model.TaskMail, keyMail, s.cfg.Scheduler.MailInterval,
		func(ctx context.Context) service.BatchResult { return s.svc.FetchAllMails(ctx, 25, s.stagger) })
	s.cleanupLogs()
}

// LogRetention 当前日志保留期；0 表示永久保留。
func (s *Scheduler) LogRetention() time.Duration {
	raw := s.svc.Settings.Get(keyLogRetention, "")
	if raw == "" {
		return defaultLogRetention
	}
	if raw == "0" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultLogRetention
	}
	return d
}

// SetLogRetention 更新日志保留期（"0" = 永久保留）。
func (s *Scheduler) SetLogRetention(raw string) error {
	if raw != "0" {
		if _, err := time.ParseDuration(raw); err != nil {
			return err
		}
	}
	return s.svc.Settings.Set(keyLogRetention, raw)
}

// CleanupLogs 按保留期删除过期日志，返回删除条数（保留期为 0 时不动作）。
func (s *Scheduler) CleanupLogs() (int64, error) {
	ret := s.LogRetention()
	if ret <= 0 {
		return 0, nil
	}
	return s.svc.TaskLogs.DeleteBefore(time.Now().Add(-ret))
}

// cleanupLogs 调度内置的每小时清理（tick 每分钟触发，这里自行限频）。
func (s *Scheduler) cleanupLogs() {
	last := s.lastRun("logclean")
	if !last.IsZero() && time.Since(last) < time.Hour {
		return
	}
	s.markRun("logclean")
	n, err := s.CleanupLogs()
	if err != nil {
		s.log.Warn("日志清理失败", "err", err)
		return
	}
	if n > 0 {
		s.log.Info("过期日志已清理", "deleted", n, "retention", s.LogRetention())
	}
}

// checkAndRun 到点则执行并记录执行时间。
func (s *Scheduler) checkAndRun(taskType, intervalKey string, defaultInterval time.Duration,
	fn func(context.Context) service.BatchResult) {
	interval := s.interval(intervalKey, defaultInterval)
	if interval <= 0 {
		return // 间隔设为 0/负数 = 禁用该任务
	}
	last := s.lastRun(taskType)
	if !last.IsZero() && time.Since(last) < interval {
		return
	}
	if !s.mu.TryLock() {
		s.log.Debug("上一批任务仍在执行，跳过本次", "task", taskType)
		return
	}
	defer s.mu.Unlock()

	s.log.Info("定时任务开始", "task", taskType)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	res := fn(ctx)
	s.markRun(taskType)
	s.log.Info("定时任务结束", "task", taskType,
		"total", res.Total, "success", res.Success, "fail", res.Fail, "skip", res.Skip)
}

func (s *Scheduler) enabled() bool {
	return s.svc.Settings.Get(keyEnabled, boolStr(s.cfg.Scheduler.Enabled)) == "true"
}

func (s *Scheduler) interval(key string, fallback time.Duration) time.Duration {
	raw := s.svc.Settings.Get(key, "")
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

func (s *Scheduler) lastRun(taskType string) time.Time {
	raw := s.svc.Settings.Get(lastRunPrefix+taskType, "")
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (s *Scheduler) markRun(taskType string) {
	_ = s.svc.Settings.Set(lastRunPrefix+taskType, time.Now().Format(time.RFC3339))
}

// Status 返回调度状态（前端展示）。
func (s *Scheduler) Status() map[string]any {
	next := func(taskType, key string, fb time.Duration) any {
		last := s.lastRun(taskType)
		iv := s.interval(key, fb)
		if last.IsZero() {
			return map[string]any{"interval": iv.String(), "last_run": nil, "due": true}
		}
		return map[string]any{
			"interval": iv.String(),
			"last_run": last.Format(time.RFC3339),
			"due":      time.Since(last) >= iv,
		}
	}
	return map[string]any{
		"enabled":   s.enabled(),
		"refresh":   next(model.TaskRefresh, keyRefresh, s.cfg.Scheduler.RefreshInterval),
		"health":    next(model.TaskHealth, keyHealth, s.cfg.Scheduler.HealthInterval),
		"keepalive": next(model.TaskKeepalive, keyKeepalive, s.cfg.Scheduler.KeepaliveInterval),
		"mail":      next(model.TaskMail, keyMail, s.cfg.Scheduler.MailInterval),
		// 日志保留期（"0" 表示永久保留）
		"log_retention": s.LogRetention().String(),
	}
}

// SetInterval 更新任务间隔（settings 表覆盖）。
func (s *Scheduler) SetInterval(taskType, raw string) error {
	d, err := time.ParseDuration(raw)
	if err != nil && raw != "0" {
		return err
	}
	_ = d
	key := map[string]string{
		model.TaskRefresh:   keyRefresh,
		model.TaskHealth:    keyHealth,
		model.TaskKeepalive: keyKeepalive,
		model.TaskMail:      keyMail,
	}[taskType]
	if key == "" {
		return nil
	}
	return s.svc.Settings.Set(key, raw)
}

// SetEnabled 开关调度。
func (s *Scheduler) SetEnabled(enabled bool) error {
	return s.svc.Settings.Set(keyEnabled, boolStr(enabled))
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
