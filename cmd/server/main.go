// outlook-manager 服务入口：加载配置、初始化 DB/日志/调度、启动 HTTP 服务。
package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"outlook-manager/internal/config"
	"outlook-manager/internal/handler"
	"outlook-manager/internal/model"
	"outlook-manager/internal/pkg"
	"outlook-manager/internal/repository"
	"outlook-manager/internal/router"
	"outlook-manager/internal/scheduler"
	"outlook-manager/internal/service"
)

const defaultConfigPath = "configs/config.yaml"

func main() {
	cfgPath := os.Getenv("OM_CONFIG")
	if cfgPath == "" {
		cfgPath = defaultConfigPath
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	log := setupLogger(cfg)
	log.Info("outlook-manager 启动", "config", cfgPath)

	db, err := repository.Open(cfg.Database.Path)
	if err != nil {
		log.Error("数据库初始化失败", "err", err)
		os.Exit(1)
	}

	svc := service.New(db, cfg, log)
	jwtSecret := ensureJWTSecret(svc, cfg)
	ensureAdmin(svc, cfg)

	sched := scheduler.New(svc, cfg, log)
	sched.Start()
	defer sched.Stop()

	h := handler.New(svc, jwtSecret, time.Duration(cfg.Auth.TokenTTLHours)*time.Hour)
	h.SetScheduler(sched)

	if os.Getenv("OM_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := router.New(h, svc, jwtSecret)
	mountStatic(r, log)

	srv := &http.Server{Addr: cfg.Addr(), Handler: r}
	go func() {
		log.Info("HTTP 服务监听", "addr", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP 服务异常退出", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("收到退出信号，正在关闭...")
}

// setupLogger 按配置初始化 slog（控制台 + 可选文件）。
func setupLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	_ = level.UnmarshalText([]byte(cfg.Log.Level))
	opts := &slog.HandlerOptions{Level: level}

	var w io.Writer = os.Stdout
	if cfg.Log.File != "" {
		if dir := filepath.Dir(cfg.Log.File); dir != "." {
			_ = os.MkdirAll(dir, 0o755)
		}
		if f, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			w = io.MultiWriter(os.Stdout, f)
		}
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

// ensureJWTSecret 优先取配置；为空则生成随机密钥并持久化到 settings 表。
func ensureJWTSecret(svc *service.Service, cfg *config.Config) string {
	if cfg.Auth.JWTSecret != "" {
		return cfg.Auth.JWTSecret
	}
	if existing := svc.Settings.Get("auth.jwt_secret", ""); existing != "" {
		return existing
	}
	secret := pkg.RandomKey(32)
	if err := svc.Settings.Set("auth.jwt_secret", secret); err != nil {
		svc.Log.Warn("JWT 密钥持久化失败，重启后所有会话将失效", "err", err)
	}
	return secret
}

// ensureAdmin 首次启动创建管理员。
// 配置未指定密码时随机生成，醒目打印一次并写入 data/initial_admin_password.txt（0600）。
func ensureAdmin(svc *service.Service, cfg *config.Config) {
	if _, err := svc.Users.ByUsername(cfg.Auth.AdminUsername); err == nil {
		return
	}

	password := cfg.Auth.AdminPassword
	generated := false
	if password == "" {
		password = pkg.RandomPassword(16)
		generated = true
	}
	hash, err := pkg.HashPassword(password)
	if err != nil {
		svc.Log.Error("管理员密码加密失败", "err", err)
		os.Exit(1)
	}
	if err := svc.Users.Create(&model.User{
		Username:     cfg.Auth.AdminUsername,
		PasswordHash: hash,
	}); err != nil {
		svc.Log.Error("创建管理员失败", "err", err)
		os.Exit(1)
	}

	if !generated {
		svc.Log.Info("已按配置创建管理员账号", "username", cfg.Auth.AdminUsername)
		return
	}

	// 首次随机密码：控制台醒目打印 + 落盘备份（仅一次，0600）
	pwdFile := "data/initial_admin_password.txt"
	_ = os.MkdirAll("data", 0o755)
	if err := os.WriteFile(pwdFile,
		[]byte(fmt.Sprintf("username: %s\npassword: %s\n", cfg.Auth.AdminUsername, password)),
		0o600); err != nil {
		svc.Log.Warn("初始密码备份文件写入失败", "err", err)
	}
	banner := fmt.Sprintf(`
============================================================
  首次启动，已生成管理员初始密码（仅显示一次）：

      用户名: %s
      密  码: %s

  已备份到: %s（权限 0600，请尽快登录后修改密码并删除该文件）
============================================================`, cfg.Auth.AdminUsername, password, pwdFile)
	fmt.Println(banner)
	svc.Log.Info("已创建管理员账号（随机初始密码）", "username", cfg.Auth.AdminUsername, "backup", pwdFile)
}
