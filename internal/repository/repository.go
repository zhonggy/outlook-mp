// Package repository 提供数据访问层（GORM/SQLite）。
package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"outlook-manager/internal/model"
)

// Open 打开（必要时创建）SQLite 数据库并自动迁移。
func Open(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	db, err := gorm.Open(sqlite.Open(path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	return db, nil
}

// AccountFilter 账号列表筛选。
type AccountFilter struct {
	Keyword string // email 模糊
	Status  string
	Tag     string
	Group   string
	Source  string
	Page    int
	Size    int
}

// AccountRepo 账号数据访问。
type AccountRepo struct{ db *gorm.DB }

func NewAccountRepo(db *gorm.DB) *AccountRepo { return &AccountRepo{db: db} }

func (r *AccountRepo) List(f AccountFilter) ([]model.Account, int64, error) {
	var (
		items []model.Account
		total int64
	)
	q := r.db.Model(&model.Account{})
	if f.Keyword != "" {
		q = q.Where("email LIKE ?", "%"+f.Keyword+"%")
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Tag != "" {
		q = q.Where("tags LIKE ?", "%"+f.Tag+"%")
	}
	if f.Group != "" {
		q = q.Where("group_name = ?", f.Group)
	}
	if f.Source != "" {
		q = q.Where("source = ?", f.Source)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size := f.Page, f.Size
	if page < 1 {
		page = 1
	}
	// 上限放宽到 50000：列表页只用小分页，大批量来自导出场景（Size 即导出条数）
	if size < 1 {
		size = 20
	}
	if size > 50000 {
		size = 50000
	}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (r *AccountRepo) All() ([]model.Account, error) {
	var items []model.Account
	err := r.db.Order("id ASC").Find(&items).Error
	return items, err
}

func (r *AccountRepo) ByID(id uint) (*model.Account, error) {
	var acc model.Account
	if err := r.db.First(&acc, id).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *AccountRepo) ByEmail(email string) (*model.Account, error) {
	var acc model.Account
	if err := r.db.Where("email = ?", email).First(&acc).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

// Upsert 按 email 存在则更新凭据，否则新建。返回 (账号, 是否新建)。
func (r *AccountRepo) Upsert(acc *model.Account) (*model.Account, bool, error) {
	existing, err := r.ByEmail(acc.Email)
	if err == gorm.ErrRecordNotFound {
		if err := r.db.Create(acc).Error; err != nil {
			return nil, false, err
		}
		return acc, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	updates := map[string]any{"updated_at": time.Now()}
	if acc.Password != "" {
		updates["password"] = acc.Password
	}
	if acc.ClientID != "" {
		updates["client_id"] = acc.ClientID
	}
	if acc.RefreshToken != "" {
		updates["refresh_token"] = acc.RefreshToken
		updates["status"] = model.StatusUnknown // 新凭据重新判定
		updates["fail_count"] = 0
	}
	if acc.Source != "" {
		updates["source"] = acc.Source
	}
	if err := r.db.Model(existing).Updates(updates).Error; err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func (r *AccountRepo) Update(id uint, fields map[string]any) error {
	fields["updated_at"] = time.Now()
	return r.db.Model(&model.Account{}).Where("id = ?", id).Updates(fields).Error
}

func (r *AccountRepo) Delete(ids []uint) error {
	return r.db.Delete(&model.Account{}, ids).Error
}

// MarkPushed 标记账号已推送到 outlookEmail。
func (r *AccountRepo) MarkPushed(id uint) error {
	now := time.Now()
	return r.db.Model(&model.Account{}).Where("id = ?", id).Update("pushed_at", &now).Error
}

// ListHealthyUnpushed 返回所有 healthy 且未推送的账号。
func (r *AccountRepo) ListHealthyUnpushed() ([]model.Account, error) {
	var accounts []model.Account
	err := r.db.Where("status = ? AND pushed_at IS NULL", model.StatusHealthy).Find(&accounts).Error
	return accounts, err
}

// DeleteByStatus 按状态删除（如一键清理失效账号），返回删除条数。
func (r *AccountRepo) DeleteByStatus(status string) (int64, error) {
	res := r.db.Where("status = ?", status).Delete(&model.Account{})
	return res.RowsAffected, res.Error
}

// DistinctGroups 全部分组名。
func (r *AccountRepo) DistinctGroups() ([]string, error) {
	var groups []string
	err := r.db.Model(&model.Account{}).Distinct().Where("group_name <> ''").Pluck("group_name", &groups).Error
	return groups, err
}

// StatusCounts 各状态计数（仪表盘）。
func (r *AccountRepo) StatusCounts() (map[string]int64, error) {
	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	if err := r.db.Model(&model.Account{}).Select("status, COUNT(*) AS cnt").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r2 := range rows {
		out[r2.Status] = r2.Cnt
	}
	return out, nil
}

// MailRepo 邮件数据访问。
type MailRepo struct{ db *gorm.DB }

func NewMailRepo(db *gorm.DB) *MailRepo { return &MailRepo{db: db} }

// UpsertMessage 按 (account_id, message_id) 幂等写入，返回是否新建。
func (r *MailRepo) UpsertMessage(m *model.MailMessage) (bool, error) {
	var existing model.MailMessage
	err := r.db.Where("account_id = ? AND message_id = ?", m.AccountID, m.MessageID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return true, r.db.Create(m).Error
	}
	if err != nil {
		return false, err
	}
	return false, r.db.Model(&existing).Updates(map[string]any{
		"subject": m.Subject, "from_addr": m.FromAddr, "body_preview": m.BodyPreview,
		"body": m.Body, "received_at": m.ReceivedAt, "is_read": m.IsRead, "code": m.Code,
		"folder": m.Folder,
	}).Error
}

func (r *MailRepo) ListByAccount(accountID uint, page, size int) ([]model.MailMessage, int64, error) {
	var (
		items []model.MailMessage
		total int64
	)
	q := r.db.Model(&model.MailMessage{}).Where("account_id = ?", accountID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	err := q.Order("received_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

func (r *MailRepo) ByID(id uint) (*model.MailMessage, error) {
	var m model.MailMessage
	if err := r.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// TaskLogRepo 任务日志。
type TaskLogRepo struct{ db *gorm.DB }

func NewTaskLogRepo(db *gorm.DB) *TaskLogRepo { return &TaskLogRepo{db: db} }

func (r *TaskLogRepo) Add(l *model.TaskLog) error { return r.db.Create(l).Error }

func (r *TaskLogRepo) List(taskType, status string, page, size int) ([]model.TaskLog, int64, error) {
	var (
		items []model.TaskLog
		total int64
	)
	q := r.db.Model(&model.TaskLog{})
	if taskType != "" {
		q = q.Where("task_type = ?", taskType)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error
	return items, total, err
}

// RecentByAccount 某账号最近日志。
func (r *TaskLogRepo) RecentByAccount(accountID uint, limit int) ([]model.TaskLog, error) {
	var items []model.TaskLog
	if limit < 1 || limit > 100 {
		limit = 20
	}
	err := r.db.Where("account_id = ?", accountID).Order("id DESC").Limit(limit).Find(&items).Error
	return items, err
}

// DeleteBefore 删除早于 cutoff 的日志，返回删除条数。
func (r *TaskLogRepo) DeleteBefore(cutoff time.Time) (int64, error) {
	res := r.db.Where("created_at < ?", cutoff).Delete(&model.TaskLog{})
	return res.RowsAffected, res.Error
}

// DeleteAll 清空全部日志，返回删除条数。
func (r *TaskLogRepo) DeleteAll() (int64, error) {
	res := r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.TaskLog{})
	return res.RowsAffected, res.Error
}

// Count 日志总条数。
func (r *TaskLogRepo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&model.TaskLog{}).Count(&n).Error
	return n, err
}

// SettingRepo 运行时设置。
type SettingRepo struct{ db *gorm.DB }

func NewSettingRepo(db *gorm.DB) *SettingRepo { return &SettingRepo{db: db} }

func (r *SettingRepo) Get(key, fallback string) string {
	var s model.Setting
	if err := r.db.Where("key = ?", key).First(&s).Error; err != nil {
		return fallback
	}
	return s.Value
}

func (r *SettingRepo) Set(key, value string) error {
	return r.db.Save(&model.Setting{Key: key, Value: value}).Error
}

func (r *SettingRepo) All() ([]model.Setting, error) {
	var items []model.Setting
	err := r.db.Find(&items).Error
	return items, err
}

// APIKeyRepo API 密钥。
type APIKeyRepo struct{ db *gorm.DB }

func NewAPIKeyRepo(db *gorm.DB) *APIKeyRepo { return &APIKeyRepo{db: db} }

func (r *APIKeyRepo) Create(k *model.APIKey) error { return r.db.Create(k).Error }

func (r *APIKeyRepo) List() ([]model.APIKey, error) {
	var items []model.APIKey
	err := r.db.Order("id DESC").Find(&items).Error
	return items, err
}

func (r *APIKeyRepo) ByKey(key string) (*model.APIKey, error) {
	var k model.APIKey
	if err := r.db.Where("key = ? AND enabled = ?", key, true).First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepo) TouchUsed(id uint) {
	now := time.Now()
	r.db.Model(&model.APIKey{}).Where("id = ?", id).Update("last_used_at", &now)
}

func (r *APIKeyRepo) Delete(id uint) error { return r.db.Delete(&model.APIKey{}, id).Error }

// UserRepo 管理员。
type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) ByUsername(name string) (*model.User, error) {
	var u model.User
	if err := r.db.Where("username = ?", name).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Create(u *model.User) error { return r.db.Create(u).Error }

func (r *UserRepo) UpdatePassword(id uint, hash string) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("password_hash", hash).Error
}
