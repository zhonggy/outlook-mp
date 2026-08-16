// Package outemail 封装 outlookEmail 项目的内部 API 客户端（登录/CSRF/导入账号）。
package outemail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client outlookEmail 内部 API 客户端。
type Client struct {
	baseURL  string
	password string
	client   *http.Client
	cookies  []*http.Cookie // 手动维护所有 cookie
}

// New 创建客户端。
func New(baseURL, password string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Login 登录获取 Session Cookie。
func (c *Client) Login() error {
	body := map[string]string{"password": c.password}
	resp, err := c.do("POST", "/login", body, nil)
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("登录失败 HTTP %d: %s", resp.StatusCode, string(b))
	}
	c.mergeCookies(resp.Cookies())
	return nil
}

// CSRFToken 获取 CSRF Token（必须先用 Login 建立 session）。
func (c *Client) CSRFToken() (string, error) {
	resp, err := c.do("GET", "/api/csrf-token", nil, nil)
	if err != nil {
		return "", fmt.Errorf("获取 CSRF Token 失败: %w", err)
	}
	defer resp.Body.Close()
	c.mergeCookies(resp.Cookies())

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
	if err := c.Login(); err != nil {
		return nil, err
	}
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
	resp, err := c.do("POST", "/api/accounts", body, headers)
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

func (c *Client) do(method, path string, body any, extraHeaders map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.baseURL+path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	// 发送所有已保存的 cookie
	for _, ck := range c.cookies {
		req.AddCookie(ck)
	}
	return c.client.Do(req)
}

// mergeCookies 合并响应中的 cookie（同名覆盖，保留其他）。
func (c *Client) mergeCookies(cookies []*http.Cookie) {
	for _, newCk := range cookies {
		found := false
		for i, oldCk := range c.cookies {
			if oldCk.Name == newCk.Name {
				c.cookies[i] = newCk
				found = true
				break
			}
		}
		if !found {
			c.cookies = append(c.cookies, newCk)
		}
	}
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