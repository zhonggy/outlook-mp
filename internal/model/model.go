// Package model 定义 GORM 数据模型与状态常量。
package model

import (
	"time"

	"gorm.io/gorm"
)

// 账号健康状态。
const (
	StatusUnknown  = "unknown"
	StatusHealthy  = "healthy"
	StatusDead     = "dead"   // refresh_token 失效（invalid_grant），需重新登录
	StatusLocked   = "locked" // 账号被微软锁定/需要验证
	StatusError    = "error"  // 网络等临时错误
)

// 账号来源。
const (
	SourceManual   = "manual"
	SourceRegister = "register" // 自动化注册器经 API 上传
	SourceImport   = "import"
)

// 任务类型。
const (
	TaskRefresh   = "refresh"
	TaskHealth    = "health"
	TaskKeepalive = "keepalive"
	TaskMail      = "mail"
)

// 任务结果。
const (
	TaskSuccess = "success"
	TaskFail    = "fail"
	TaskSkip    = "skip"
)

// Account 邮箱账号（含 OAuth 凭据与健康状态）。
type Account struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Email          string         `gorm:"uniqueIndex;size:191;not null" json:"email"`
	Password       string         `gorm:"size:255" json:"password,omitempty"`
	ClientID       string         `gorm:"size:64" json:"client_id"`
	RefreshToken   string         `gorm:"type:text" json:"refresh_token,omitempty"`
	AccessToken    string         `gorm:"type:text" json:"-"` // 短期凭据，不导出
	TokenExpiresAt *time.Time     `json:"token_expires_at,omitempty"`

	Status       string `gorm:"size:16;index;default:unknown" json:"status"`
	StatusReason string `gorm:"size:512" json:"status_reason"`
	FailCount    int    `gorm:"default:0" json:"fail_count"`

	Tags      string `gorm:"size:255;index" json:"tags"` // 逗号分隔
	GroupName string `gorm:"size:64;index" json:"group_name"`
	Remark    string `gorm:"size:512" json:"remark"`
	Proxy     string `gorm:"size:255" json:"proxy"` // 账号级代理；空=全局
	Source    string `gorm:"size:32;default:manual" json:"source"`

	LastRefreshAt   *time.Time `json:"last_refresh_at,omitempty"`
	LastCheckAt     *time.Time `json:"last_check_at,omitempty"`
	LastKeepaliveAt *time.Time `json:"last_keepalive_at,omitempty"`
	LastMailAt      *time.Time `json:"last_mail_at,omitempty"`
	PushedAt        *time.Time `json:"pushed_at,omitempty"`  // 推送到 outlookEmail 的时间

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// MailMessage 拉取缓存的邮件。
type MailMessage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AccountID   uint      `gorm:"index:idx_mail_acc_msg,unique;not null" json:"account_id"`
	MessageID   string    `gorm:"size:191;index:idx_mail_acc_msg,unique;not null" json:"message_id"`
	Subject     string    `gorm:"size:512" json:"subject"`
	FromAddr    string    `gorm:"size:255" json:"from_addr"`
	BodyPreview string    `gorm:"size:512" json:"body_preview"`
	Body        string    `gorm:"type:text" json:"body,omitempty"`
	ReceivedAt  time.Time `gorm:"index" json:"received_at"`
	IsRead      bool      `json:"is_read"`
	Folder      string    `gorm:"size:16;default:inbox" json:"folder"` // inbox / junk（垃圾邮件文件夹）
	Code        string    `gorm:"size:16" json:"code,omitempty"` // 提取的验证码
	CreatedAt   time.Time `json:"created_at"`
}

// TaskLog 任务执行日志。
type TaskLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TaskType   string    `gorm:"size:16;index" json:"task_type"`
	AccountID  *uint     `gorm:"index" json:"account_id,omitempty"`
	Email      string    `gorm:"size:191;index" json:"email"`
	Status     string    `gorm:"size:16;index" json:"status"`
	Message    string    `gorm:"size:1024" json:"message"`
	DurationMs int64     `json:"duration_ms"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

// User 系统管理员。
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// APIKey 自动化对接密钥（外部系统上传账号用）。
type APIKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"size:64;not null" json:"name"`
	Key        string     `gorm:"uniqueIndex;size:64;not null" json:"key"`
	Enabled    bool       `gorm:"default:true" json:"enabled"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Setting 运行时设置（覆盖 config.yaml 的调度配置等）。
type Setting struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `gorm:"size:1024" json:"value"`
}

// AllModels 供 AutoMigrate 使用。
func AllModels() []any {
	return []any{
		&Account{}, &MailMessage{}, &TaskLog{}, &User{}, &APIKey{}, &Setting{},
	}
}
