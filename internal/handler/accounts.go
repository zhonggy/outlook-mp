package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"outlook-manager/internal/model"
	"outlook-manager/internal/repository"
	"outlook-manager/internal/service"
)

func parseFilter(c *gin.Context) repository.AccountFilter {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	return repository.AccountFilter{
		Keyword: c.Query("keyword"),
		Status:  c.Query("status"),
		Tag:     c.Query("tag"),
		Group:   c.Query("group"),
		Source:  c.Query("source"),
		Page:    page,
		Size:    size,
	}
}

// ListAccounts GET /accounts
func (h *Handler) ListAccounts(c *gin.Context) {
	items, total, err := h.svc.Accounts.List(parseFilter(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// GetAccount GET /accounts/:id
func (h *Handler) GetAccount(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	acc, err := h.svc.Accounts.ByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
		return
	}
	logs, _ := h.svc.TaskLogs.RecentByAccount(acc.ID, 20)
	c.JSON(http.StatusOK, gin.H{"account": acc, "recent_logs": logs})
}

type createAccountReq struct {
	Email        string `json:"email" binding:"required"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
	Tags         string `json:"tags"`
	GroupName    string `json:"group_name"`
	Remark       string `json:"remark"`
	Proxy        string `json:"proxy"`
}

// CreateAccount POST /accounts
func (h *Handler) CreateAccount(c *gin.Context) {
	var req createAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email 必填"})
		return
	}
	res := h.svc.ImportAccounts([]service.ImportItem{{
		Email: req.Email, Password: req.Password, ClientID: req.ClientID,
		RefreshToken: req.RefreshToken, Tags: req.Tags, GroupName: req.GroupName,
		Remark: req.Remark, Source: model.SourceManual,
	}}, model.SourceManual)
	if req.Proxy != "" {
		if acc, err := h.svc.Accounts.ByEmail(strings.ToLower(req.Email)); err == nil {
			_ = h.svc.Accounts.Update(acc.ID, map[string]any{"proxy": req.Proxy})
		}
	}
	c.JSON(http.StatusOK, res)
}

type updateAccountReq struct {
	Password  *string `json:"password"`
	Tags      *string `json:"tags"`
	GroupName *string `json:"group_name"`
	Remark    *string `json:"remark"`
	Proxy     *string `json:"proxy"`
}

// UpdateAccount PUT /accounts/:id
func (h *Handler) UpdateAccount(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if _, err := h.svc.Accounts.ByID(uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
		return
	}
	var req updateAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	fields := map[string]any{}
	if req.Password != nil {
		fields["password"] = *req.Password
	}
	if req.Tags != nil {
		fields["tags"] = *req.Tags
	}
	if req.GroupName != nil {
		fields["group_name"] = *req.GroupName
	}
	if req.Remark != nil {
		fields["remark"] = *req.Remark
	}
	if req.Proxy != nil {
		fields["proxy"] = *req.Proxy
	}
	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无可更新字段"})
		return
	}
	if err := h.svc.Accounts.Update(uint(id), fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已更新"})
}

// DeleteAccounts DELETE /accounts/:id 与 POST /accounts/batch-delete
func (h *Handler) DeleteAccount(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Accounts.Delete([]uint{uint(id)}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

type batchDeleteReq struct {
	IDs []uint `json:"ids" binding:"required"`
}

func (h *Handler) BatchDeleteAccounts(c *gin.Context) {
	var req batchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids 必填"})
		return
	}
	if err := h.svc.Accounts.Delete(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除", "count": len(req.IDs)})
}

// ImportAccounts POST /accounts/import
// 支持三种载荷：JSON 数组、text/plain（---- 或 管道格式）、multipart 文件。
func (h *Handler) ImportAccounts(c *gin.Context) {
	ct := c.ContentType()
	var items []service.ImportItem

	switch {
	case strings.HasPrefix(ct, "application/json"):
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
			return
		}
		var single struct {
			Text string `json:"text"`
		}
		items, err = service.ParseImportJSON(body)
		if err != nil {
			// 兼容 {"text": "..."} 包裹
			if jerr := jsonUnmarshal(body, &single); jerr == nil && single.Text != "" {
				items = service.ParseImportText(single.Text)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "JSON 解析失败: " + err.Error()})
				return
			}
		}
	default: // text/plain 等按文本解析
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
			return
		}
		items = service.ParseImportText(string(body))
	}

	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未解析到有效账号"})
		return
	}
	res := h.svc.ImportAccounts(items, model.SourceImport)
	c.JSON(http.StatusOK, res)
}

func jsonUnmarshal(data []byte, v any) error {
	return jsonUnmarshalStd(data, v)
}

// ExportAccounts GET /accounts/export?format=json|txt|csv&limit=N（limit 缺省/0 为全部）
func (h *Handler) ExportAccounts(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	f := parseFilter(c)
	f.Page, f.Size = 1, limit
	content, mime, err := h.svc.ExportAccounts(f, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=accounts."+format)
	c.Data(http.StatusOK, mime, []byte(content))
}

type deleteByStatusReq struct {
	Status string `json:"status" binding:"required"`
}

// DeleteAccountsByStatus POST /accounts/delete-by-status（一键清理某状态账号，如失效）
func (h *Handler) DeleteAccountsByStatus(c *gin.Context) {
	var req deleteByStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status 必填"})
		return
	}
	switch req.Status {
	case model.StatusDead, model.StatusLocked, model.StatusError, model.StatusUnknown, model.StatusHealthy:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法状态"})
		return
	}
	n, err := h.svc.Accounts.DeleteByStatus(req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除", "count": n})
}

// IngestAccounts POST /ingest/accounts（API Key 认证，外部自动化系统上传）
func (h *Handler) IngestAccounts(c *gin.Context) {
	var items []service.ImportItem
	if err := c.ShouldBindJSON(&items); err != nil {
		// 兼容单条上传
		var single service.ImportItem
		if err2 := c.ShouldBindJSON(&single); err2 != nil || single.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "载荷须为账号数组或单个账号对象"})
			return
		}
		items = []service.ImportItem{single}
	}
	res := h.svc.ImportAccounts(items, model.SourceRegister)
	c.JSON(http.StatusOK, res)
}

// RefreshAccount POST /accounts/:id/refresh
func (h *Handler) RefreshAccount(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	acc, err := h.svc.Accounts.ByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
		return
	}
	res := h.svc.RefreshAccount(c.Request.Context(), acc)
	status := http.StatusOK
	if res.Status == model.TaskFail {
		status = http.StatusBadGateway
	}
	c.JSON(status, res)
}

// CheckAccount POST /accounts/:id/check
func (h *Handler) CheckAccount(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	acc, err := h.svc.Accounts.ByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "账号不存在"})
		return
	}
	res := h.svc.CheckHealth(c.Request.Context(), acc)
	status := http.StatusOK
	if res.Status == model.TaskFail {
		status = http.StatusBadGateway
	}
	c.JSON(status, res)
}

// RefreshAllAccounts POST /accounts/refresh-all（同步执行并返回汇总）
func (h *Handler) RefreshAllAccounts(c *gin.Context) {
	res := h.svc.RefreshAll(c.Request.Context(), "", 0)
	c.JSON(http.StatusOK, res)
}

// CheckAllAccounts POST /accounts/check-all
func (h *Handler) CheckAllAccounts(c *gin.Context) {
	res := h.svc.CheckAll(c.Request.Context(), "", 0)
	c.JSON(http.StatusOK, res)
}

// ListGroups GET /accounts/groups
func (h *Handler) ListGroups(c *gin.Context) {
	groups, err := h.svc.Accounts.DistinctGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}
