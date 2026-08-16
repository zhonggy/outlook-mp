// Package outemail 封装 outlookEmail 项目的内部 API 客户端（登录/CSRF/导入账号）。
package outemail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// Client outlookEmail 内部 API 客户端。
type Client struct {
	baseURL  string
	password string
	client   *http.Client
}

// New 创建客户端。
func New(baseURL, password string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		password: password,
		client:   &http.Client{Jar: jar, Timeout: 30 * time.Second},
	}
}

// Login 登录获取 Session Cookie。
func (c *Client) Login() error {
	body := map[string]string{"password": c.password}
	resp, err := c.post("/login", body, nil)
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("登录失败 HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// CSRFToken 获取 CSRF Token。
func (c *Client) CSRFToken() (string, error) {
	resp, err := c.get("/api/csrf-token")
	if err != nil {
		return "", fmt.Errorf("获取 CSRF Token 失败: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		CSRFToken    string `json:"csrf_token"`
		CSRFDisabled bool   `json:"csrf_disabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析 CSRF Token 失败: %w", err)
	}
	if result.CSRFDisabled {
		return "", nil
	}
	return result.CSRFToken, nil
}

// ImportAccounts 批量导入账号（格式：邮箱----密码----ClientID----RefreshToken）。
// groupID 为 outlookEmail 中的分组 ID。
func (c *Client) ImportAccounts(accountString string, groupID int) (*ImportResult, error) {
	csrf, err := c.CSRFToken()
	if err != nil {
		return nil, err
	}
	headers := map[string]string{}
	if csrf != "" {
		headers["X-CSRFToken"] = csrf
	}

	body := map[string]any{
		"account_string": accountString,
		"group_id":       groupID,
		"account_format": "client_id_refresh_token",
		"provider":       "outlook",
		"status":         "active",
	}
	resp, err := c.post("/api/accounts", body, headers)
	if err != nil {
		return nil, fmt.Errorf("导入账号失败: %w", err)
	}
	defer resp.Body.Close()

	var result ImportResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析导入结果失败: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("导入失败: %s", result.Error)
	}
	return &result, nil
}

// Test 测试连接（登录 + 获取 CSRF Token）。
func (c *Client) Test() error {
	if err := c.Login(); err != nil {
		return err
	}
	_, err := c.CSRFToken()
	return err
}

// ---- 内部 ----

func (c *Client) get(path string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", c.baseURL+path, nil)
	req.Header.Set("Accept", "application/json")
	return c.client.Do(req)
}

func (c *Client) post(path string, body any, extraHeaders map[string]string) (*http.Response, error) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	return c.client.Do(req)
}

// ImportResult 导入结果。
type ImportResult struct {
	Success      bool   `json:"success"`
	Error        string `json:"error"`
	AddedCount   int    `json:"added_count"`
	SkippedCount int    `json:"skipped_count"`
	InvalidCount int    `json:"invalid_count"`
	TaggedCount  int    `json:"tagged_count"`
}

// 避免 unused import
var _ = url.Parse