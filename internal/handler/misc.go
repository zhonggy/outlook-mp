package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"outlook-manager/internal/model"
	"outlook-manager/internal/pkg"
	"outlook-manager/internal/service"
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

// ---- OutlookRegister 对接 ----

type importRegisterReq struct {
	Text string `json:"text" binding:"required"`
}

// ImportFromRegister POST /settings/import-from-register
// 接受 OutlookRegister 输出的 oauth2.txt 格式（email----password----client_id----refresh_token），
// 批量导入账号，来源标记为 register。
func (h *Handler) ImportFromRegister(c *gin.Context) {
	var req importRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请粘贴 oauth2.txt 内容"})
		return
	}
	items := service.ParseImportText(req.Text)
	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未解析到有效账号，请确认格式为 email----password----client_id----refresh_token"})
		return
	}
	res := h.svc.ImportAccounts(items, model.SourceRegister)
	c.JSON(http.StatusOK, res)
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
