// Package service 业务逻辑层：账号、token 刷新、健康检测、保活、收信。
package service

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"outlook-manager/internal/config"
	"outlook-manager/internal/model"
	"outlook-manager/internal/msgraph"
	"outlook-manager/internal/outemail"
	"outlook-manager/internal/repository"
)

// Service 聚合数据访问、微软客户端与配置，供 handler/scheduler 调用。
type Service struct {
	DB       *gorm.DB
	Cfg      *config.Config
	MS       *msgraph.Client
	Log      *slog.Logger
	Accounts *repository.AccountRepo
	Mails    *repository.MailRepo
	TaskLogs *repository.TaskLogRepo
	Settings *repository.SettingRepo
	APIKeys  *repository.APIKeyRepo
	Users    *repository.UserRepo
}

// New 组装业务服务。
func New(db *gorm.DB, cfg *config.Config, log *slog.Logger) *Service {
	return &Service{
		DB:       db,
		Cfg:      cfg,
		MS:       msgraph.New(cfg.Microsoft.TokenURL, cfg.Microsoft.GraphURL, cfg.Microsoft.Scope),
		Log:      log,
		Accounts: repository.NewAccountRepo(db),
		Mails:    repository.NewMailRepo(db),
		TaskLogs: repository.NewTaskLogRepo(db),
		Settings: repository.NewSettingRepo(db),
		APIKeys:  repository.NewAPIKeyRepo(db),
		Users:    repository.NewUserRepo(db),
	}
}

// proxyFor 为账号选择代理：账号级优先，其次全局。
func (s *Service) proxyFor(acc *model.Account) string {
	if acc.Proxy != "" {
		return acc.Proxy
	}
	if s.Cfg.Proxy.Enabled {
		return s.Cfg.Proxy.URL
	}
	return ""
}

// logTask 落一条任务执行日志（失败不阻塞主流程）。
func (s *Service) logTask(taskType string, acc *model.Account, status, message string, start time.Time) {
	entry := &model.TaskLog{
		TaskType:   taskType,
		Status:     status,
		Message:    message,
		DurationMs: time.Since(start).Milliseconds(),
		CreatedAt:  time.Now(),
	}
	if acc != nil {
		entry.AccountID = &acc.ID
		entry.Email = acc.Email
	}
	if err := s.TaskLogs.Add(entry); err != nil {
		s.Log.Warn("写任务日志失败", "err", err)
	}
}

// tryAutoPush 健康检测成功后自动推送到 outlookEmail（启用了才执行，失败不阻塞）。
func (s *Service) tryAutoPush(acc *model.Account) {
	enabled := s.Settings.Get("outlook_email.enabled", "false") == "true"
	if !enabled {
		return
	}
	// 已推送过的跳过，防止每轮健康检测重复推送
	if fresh, err := s.Accounts.ByID(acc.ID); err == nil && fresh.PushedAt != nil {
		return
	}
	url := s.Settings.Get("outlook_email.url", "")
	pwd := s.Settings.Get("outlook_email.password", "")
	if url == "" || pwd == "" {
		return
	}
	groupID := 1
	fmt.Sscanf(s.Settings.Get("outlook_email.group_id", "1"), "%d", &groupID)
	if groupID <= 0 {
		groupID = 1
	}

	text := fmt.Sprintf("%s----%s----%s----%s", acc.Email, acc.Password, acc.ClientID, acc.RefreshToken)
	client := outemail.New(url, pwd)
	if _, err := client.ImportAccounts(text, groupID); err != nil {
		s.Log.Warn("自动推送到 outlookEmail 失败", "email", acc.Email, "err", err)
		return
	}
	_ = s.Accounts.MarkPushed(acc.ID)
	s.Log.Info("已推送到 outlookEmail", "email", acc.Email)
}
