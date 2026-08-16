package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"outlook-manager/internal/model"
	"outlook-manager/internal/outemail"
	"outlook-manager/internal/pkg"
)

func jsonUnmarshalStd(data []byte, v any) error { return json.Unmarshal(data, v) }

// ---- 邮件 ----

// ListMails GET /accounts/:id/mails
func (h *Handler) ListMails(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	items, total, err := h.svc.Mails.ListByAccount(uint(id), page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// FetchMails POST /accounts/:id/mails/fetch
func (h *Handler) FetchMails(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	acc, err := h.svc.Accounts.ByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
		return
	}
	newCount, inboxN, junkN, err := h.svc.FetchMails(c.Request.Context(), acc, 25)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"new": newCount, "inbox": inboxN, "junk": junkN})
}

// GetMail GET /mails/:id（按需拉全文）
func (h *Handler) GetMail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	m, err := h.svc.FetchMailBody(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, m)
}

// ---- 任务与调度 ----

// ListTaskLogs GET /tasks/logs
func (h *Handler) ListTaskLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	items, total, err := h.svc.TaskLogs.List(c.Query("type"), c.Query("status"), page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// ---- 统计 ----

// Dashboard GET /stats/dashboard
func (h *Handler) Dashboard(c *gin.Context) {
	counts, err := h.svc.Accounts.StatusCounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var total int64
	for _, v := range counts {
		total += v
	}
	var mailCount int64
	h.svc.DB.Model(&model.MailMessage{}).Count(&mailCount)
	var todayLogs int64
	h.svc.DB.Model(&model.TaskLog{}).
		Where("created_at >= date('now', 'start of day')").Count(&todayLogs)
	c.JSON(http.StatusOK, gin.H{
		"total_accounts": total,
		"status_counts":  counts,
		"mail_count":     mailCount,
		"today_tasks":    todayLogs,
	})
}

// ---- 设置与 API Key ----

// outlookEmail 对接配置（持久化到 settings 表）。
func (h *Handler) outlookEmailConfig() (enabled bool, baseURL, password string, groupID int) {
	baseURL = h.svc.Settings.Get("outlook_email.url", "")
	password = h.svc.Settings.Get("outlook_email.password", "")
	enabled = h.svc.Settings.Get("outlook_email.enabled", "false") == "true"
	gid := h.svc.Settings.Get("outlook_email.group_id", "1")
	fmt.Sscanf(gid, "%d", &groupID)
	if groupID <= 0 {
		groupID = 1
	}
	return
}

// GetOutlookEmailConfig GET /settings/outlook-email
func (h *Handler) GetOutlookEmailConfig(c *gin.Context) {
	enabled, url, pwd, groupID := h.outlookEmailConfig()
	c.JSON(http.StatusOK, gin.H{
		"enabled":  enabled,
		"url":      url,
		"password": pwd,
		"group_id": groupID,
	})
}

// SetOutlookEmailConfig PUT /settings/outlook-email
func (h *Handler) SetOutlookEmailConfig(c *gin.Context) {
	var req struct {
		Enabled  *bool   `json:"enabled"`
		URL      *string `json:"url"`
		Password *string `json:"password"`
		GroupID  *int    `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.Enabled != nil {
		v := "false"
		if *req.Enabled {
			v = "true"
		}
		_ = h.svc.Settings.Set("outlook_email.enabled", v)
	}
	if req.URL != nil {
		_ = h.svc.Settings.Set("outlook_email.url", strings.TrimRight(*req.URL, "/"))
	}
	if req.Password != nil {
		_ = h.svc.Settings.Set("outlook_email.password", *req.Password)
	}
	if req.GroupID != nil {
		_ = h.svc.Settings.Set("outlook_email.group_id", fmt.Sprintf("%d", *req.GroupID))
	}
	c.JSON(http.StatusOK, gin.H{"message": "已保存"})
}

// TestOutlookEmail POST /settings/outlook-email/test
func (h *Handler) TestOutlookEmail(c *gin.Context) {
	_, url, pwd, _ := h.outlookEmailConfig()
	if url == "" || pwd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 outlookEmail 地址和密码"})
		return
	}
	client := outemail.New(url, pwd)
	if err := client.Test(); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "detail": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "detail": "连接成功"})
}

// PushToOutlookEmail POST /settings/outlook-email/push
func (h *Handler) PushToOutlookEmail(c *gin.Context) {
	_, url, pwd, groupID := h.outlookEmailConfig()
	if url == "" || pwd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置 outlookEmail 地址和密码"})
		return
	}

	// 查询所有 healthy 且未推送的账号
	accounts, err := h.svc.Accounts.ListHealthyUnpushed()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(accounts) == 0 {
		c.JSON(http.StatusOK, gin.H{"ok": true, "detail": "没有需要推送的账号", "pushed": 0})
		return
	}

	// 拼成 outlookEmail 导入格式
	var lines []string
	for _, a := range accounts {
		lines = append(lines, fmt.Sprintf("%s----%s----%s----%s", a.Email, a.Password, a.ClientID, a.RefreshToken))
	}
	text := strings.Join(lines, "\n")

	client := outemail.New(url, pwd)
	result, err := client.ImportAccounts(text, groupID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "detail": err.Error()})
		return
	}

	// 标记已推送
	for _, a := range accounts {
		_ = h.svc.Accounts.MarkPushed(a.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"pushed":  len(accounts),
		"added":   result.AddedCount,
		"skipped": result.SkippedCount,
		"invalid": result.InvalidCount,
	})
}

// ListAPIKeys GET /apikeys
func (h *Handler) ListAPIKeys(c *gin.Context) {
	items, err := h.svc.APIKeys.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type createAPIKeyReq struct {
	Name string `json:"name" binding:"required"`
}

// CreateAPIKey POST /apikeys
func (h *Handler) CreateAPIKey(c *gin.Context) {
	var req createAPIKeyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 必填"})
		return
	}
	key := &model.APIKey{Name: req.Name, Key: "omk_" + pkg.RandomKey(24), Enabled: true}
	if err := h.svc.APIKeys.Create(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, key)
}

// DeleteAPIKey DELETE /apikeys/:id
func (h *Handler) DeleteAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.APIKeys.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
