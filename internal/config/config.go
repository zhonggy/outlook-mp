// Package config 负责加载与保存系统配置（config.yaml + 环境变量覆盖）。
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 系统根配置。
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Auth      AuthConfig      `yaml:"auth"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Proxy     ProxyConfig     `yaml:"proxy"`
	Microsoft MicrosoftConfig `yaml:"microsoft"`
	Log       LogConfig       `yaml:"log"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AuthConfig struct {
	JWTSecret     string `yaml:"jwt_secret"`
	TokenTTLHours int    `yaml:"token_ttl_hours"`
	AdminUsername string `yaml:"admin_username"`
	AdminPassword string `yaml:"admin_password"`
}

// SchedulerConfig 各定时任务周期（可被 settings 表运行时覆盖）。
type SchedulerConfig struct {
	Enabled           bool          `yaml:"enabled"`
	RefreshInterval   time.Duration `yaml:"refresh_interval"`
	HealthInterval    time.Duration `yaml:"health_interval"`
	KeepaliveInterval time.Duration `yaml:"keepalive_interval"`
	MailInterval      time.Duration `yaml:"mail_interval"`
}

type ProxyConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"` // 如 http://127.0.0.1:7897
}

type MicrosoftConfig struct {
	TokenURL string `yaml:"token_url"`
	GraphURL string `yaml:"graph_url"`
	Scope    string `yaml:"scope"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// Default 内置默认配置。
func Default() *Config {
	return &Config{
		Server:   ServerConfig{Host: "0.0.0.0", Port: 18327},
		Database: DatabaseConfig{Path: "data/outlook-manager.db"},
		Auth: AuthConfig{
			JWTSecret:     "",
			TokenTTLHours: 72,
			AdminUsername: "admin",
			AdminPassword: "", // 留空 = 首次启动随机生成并打印一次
		},
		Scheduler: SchedulerConfig{
			Enabled:           true,
			RefreshInterval:   12 * time.Hour,
			HealthInterval:    6 * time.Hour,
			KeepaliveInterval: 24 * time.Hour,
			MailInterval:      30 * time.Minute,
		},
		Proxy: ProxyConfig{Enabled: false, URL: ""},
		Microsoft: MicrosoftConfig{
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			GraphURL: "https://graph.microsoft.com/v1.0",
			Scope:    "https://graph.microsoft.com/.default offline_access",
		},
		Log: LogConfig{Level: "info", File: "data/outlook-manager.log"},
	}
}

// Load 读取配置文件；文件不存在时返回默认配置（并写入一份默认文件）。
func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if werr := Save(path, cfg); werr != nil {
				return nil, fmt.Errorf("写入默认配置失败: %w", werr)
			}
			cfg.applyEnv()
			return cfg, nil
		}
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	cfg.applyEnv()
	return cfg, nil
}

// Save 把配置写回文件。
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// applyEnv 支持 OM_* 环境变量覆盖关键配置。
func (c *Config) applyEnv() {
	if v := os.Getenv("OM_JWT_SECRET"); v != "" {
		c.Auth.JWTSecret = v
	}
	if v := os.Getenv("OM_ADMIN_PASSWORD"); v != "" {
		c.Auth.AdminPassword = v
	}
	if v := os.Getenv("OM_PROXY"); v != "" {
		c.Proxy.Enabled = true
		c.Proxy.URL = v
	}
	if v := os.Getenv("OM_DB_PATH"); v != "" {
		c.Database.Path = v
	}
	if v := os.Getenv("OM_PORT"); v != "" {
		var p int
		if _, err := fmt.Sscanf(v, "%d", &p); err == nil && p > 0 {
			c.Server.Port = p
		}
	}
}

// Addr 服务监听地址。
func (c *Config) Addr() string {
	host := strings.TrimSpace(c.Server.Host)
	if host == "" {
		host = "0.0.0.0"
	}
	return fmt.Sprintf("%s:%d", host, c.Server.Port)
}
