// Package handler HTTP API 处理器。
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"outlook-manager/internal/pkg"
	"outlook-manager/internal/scheduler"
	"outlook-manager/internal/service"
)

// Handler 聚合业务服务。
type Handler struct {
	svc       *service.Service
	jwtSecret string
	jwtTTL    time.Duration
	sched     *scheduler.Scheduler
}

func New(svc *service.Service, jwtSecret string, jwtTTL time.Duration) *Handler {
	return &Handler{svc: svc, jwtSecret: jwtSecret, jwtTTL: jwtTTL}
}

// SetScheduler 注入调度器（调度配置 API 用）。
func (h *Handler) SetScheduler(s *scheduler.Scheduler) { h.sched = s }

// GetSchedule GET /tasks/schedule
func (h *Handler) GetSchedule(c *gin.Context) {
	c.JSON(http.StatusOK, h.sched.Status())
}

type setScheduleReq struct {
	Enabled           *bool  `json:"enabled"`
	RefreshInterval   string `json:"refresh_interval"`
	HealthInterval    string `json:"health_interval"`
	KeepaliveInterval string `json:"keepalive_interval"`
	MailInterval      string `json:"mail_interval"`
	LogRetention      string `json:"log_retention"`
}

// SetSchedule PUT /tasks/schedule
func (h *Handler) SetSchedule(c *gin.Context) {
	var req setScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.Enabled != nil {
		if err := h.sched.SetEnabled(*req.Enabled); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	for taskType, raw := range map[string]string{
		"refresh":   req.RefreshInterval,
		"health":    req.HealthInterval,
		"keepalive": req.KeepaliveInterval,
		"mail":      req.MailInterval,
	} {
		if raw == "" {
			continue
		}
		if err := h.sched.SetInterval(taskType, raw); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": taskType + " 间隔格式非法（示例 12h/30m）"})
			return
		}
	}
	if req.LogRetention != "" {
		if err := h.sched.SetLogRetention(req.LogRetention); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "日志保留期格式非法（示例 720h/0）"})
			return
		}
	}
	c.JSON(http.StatusOK, h.sched.Status())
}

// CleanupTaskLogs POST /tasks/logs/cleanup
// 默认按当前保留期清理过期日志；body {"all": true} 时清空全部。
func (h *Handler) CleanupTaskLogs(c *gin.Context) {
	var req struct {
		All bool `json:"all"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
			return
		}
	}
	var (
		n   int64
		err error
	)
	if req.All {
		n, err = h.svc.TaskLogs.DeleteAll()
	} else {
		n, err = h.sched.CleanupLogs()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	total, _ := h.svc.TaskLogs.Count()
	c.JSON(http.StatusOK, gin.H{"deleted": n, "remaining": total})
}

// ---- 认证 ----

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名与密码必填"})
		return
	}
	user, err := h.svc.Users.ByUsername(req.Username)
	if err != nil || !pkg.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, err := pkg.GenerateToken(h.jwtSecret, user.ID, user.Username, h.jwtTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签发令牌失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"username": user.Username,
		"expires":  time.Now().Add(h.jwtTTL).Unix(),
	})
}

func (h *Handler) Profile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"id":       c.GetUint("userID"),
		"username": c.GetString("username"),
	})
}

type changePwdReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req changePwdReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误（新密码至少 6 位）"})
		return
	}
	user, err := h.svc.Users.ByUsername(c.GetString("username"))
	if err != nil || !pkg.CheckPassword(req.OldPassword, user.PasswordHash) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原密码错误"})
		return
	}
	hash, err := pkg.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	if err := h.svc.Users.UpdatePassword(user.ID, hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码已更新"})
}
