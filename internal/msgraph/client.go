// Package msgraph 封装微软 OAuth token 刷新与 Graph API 调用。
package msgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 错误类别：区分「凭据已死」与「临时故障」。
var (
	// ErrInvalidGrant refresh_token 已失效，需重新登录获取。
	ErrInvalidGrant = errors.New("invalid_grant")
	// ErrTemporarilyUnavailable 网络/服务端临时故障。
	ErrTemporarilyUnavailable = errors.New("temporary_unavailable")
)

// TokenResult 刷新结果。
type TokenResult struct {
	AccessToken  string
	RefreshToken string // 微软会轮换 refresh_token，必须回写
	ExpiresIn    int    // 秒
}

// Message Graph 邮件结构（精简）。
type Message struct {
	ID               string    `json:"id"`
	Subject          string    `json:"subject"`
	BodyPreview      string    `json:"bodyPreview"`
	ReceivedDateTime time.Time `json:"receivedDateTime"`
	IsRead           bool      `json:"isRead"`
	From             struct {
		EmailAddress struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		} `json:"emailAddress"`
	} `json:"from"`
	Body struct {
		ContentType string `json:"contentType"`
		Content     string `json:"content"`
	} `json:"body"`
}

// Client 微软 API 客户端。
type Client struct {
	tokenURL string
	graphURL string
	scope    string
}

func New(tokenURL, graphURL, scope string) *Client {
	return &Client{tokenURL: tokenURL, graphURL: graphURL, scope: scope}
}

// httpClient 按代理构造带超时的 HTTP 客户端。
func httpClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("代理地址非法: %w", err)
		}
		transport.Proxy = http.ProxyURL(u)
	}
	return &http.Client{Transport: transport, Timeout: timeout}, nil
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// RefreshAccessToken 用 refresh_token 换新 access_token（与新的 refresh_token）。
func (c *Client) RefreshAccessToken(ctx context.Context, clientID, refreshToken, proxyURL string) (*TokenResult, error) {
	if clientID == "" || refreshToken == "" {
		return nil, fmt.Errorf("缺少 client_id 或 refresh_token")
	}
	form := url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {c.scope},
	}
	hc, err := httpClient(proxyURL, 30*time.Second)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("解析 token 响应失败: %w (HTTP %d)", err, resp.StatusCode)
	}
	if tr.Error != "" {
		desc := tr.ErrorDescription
		if tr.Error == "invalid_grant" {
			return nil, fmt.Errorf("%w: %s", ErrInvalidGrant, trimDesc(desc))
		}
		if tr.Error == "temporarily_unavailable" || resp.StatusCode >= 500 {
			return nil, fmt.Errorf("%w: %s", ErrTemporarilyUnavailable, trimDesc(desc))
		}
		return nil, fmt.Errorf("token 错误 %s: %s", tr.Error, trimDesc(desc))
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("%w: HTTP %d", ErrTemporarilyUnavailable, resp.StatusCode)
		}
		return nil, fmt.Errorf("token 请求失败 HTTP %d", resp.StatusCode)
	}
	if tr.AccessToken == "" {
		return nil, errors.New("token 响应缺少 access_token")
	}
	if tr.RefreshToken == "" {
		tr.RefreshToken = refreshToken // 未轮换则沿用旧值
	}
	return &TokenResult{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		ExpiresIn:    tr.ExpiresIn,
	}, nil
}

// GetMe 拉取账号资料（保活 + 健康探测）。
func (c *Client) GetMe(ctx context.Context, accessToken, proxyURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.graphURL+"/me?$select=id,mail,userPrincipalName", nil)
	if err != nil {
		return err
	}
	return c.doGraph(req, accessToken, proxyURL, nil)
}

// listQuery 构造邮件列表查询串（url.Values.Encode 后 + 号还原为 %20，Graph 更兼容）。
func listQuery(top int) string {
	if top < 1 || top > 100 {
		top = 25
	}
	q := url.Values{}
	q.Set("$top", strconv.Itoa(top))
	q.Set("$orderby", "receivedDateTime desc")
	q.Set("$select", "id,subject,from,receivedDateTime,bodyPreview,isRead")
	return strings.ReplaceAll(q.Encode(), "+", "%20")
}

// ListMessages 拉收件箱（按时间倒序）。
func (c *Client) ListMessages(ctx context.Context, accessToken string, top int, proxyURL string) ([]Message, error) {
	u := c.graphURL + "/me/messages?" + listQuery(top)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Value []Message `json:"value"`
	}
	if err := c.doGraph(req, accessToken, proxyURL, &out); err != nil {
		return nil, err
	}
	return out.Value, nil
}

// ListMessagesInFolder 拉指定文件夹（well-known 名：inbox / junkemail / deleteditems 等）。
func (c *Client) ListMessagesInFolder(ctx context.Context, accessToken, folder string, top int, proxyURL string) ([]Message, error) {
	u := c.graphURL + "/me/mailFolders/" + url.PathEscape(folder) + "/messages?" + listQuery(top)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Value []Message `json:"value"`
	}
	if err := c.doGraph(req, accessToken, proxyURL, &out); err != nil {
		return nil, err
	}
	return out.Value, nil
}

// GetMessage 拉单封邮件全文。
func (c *Client) GetMessage(ctx context.Context, accessToken, messageID, proxyURL string) (*Message, error) {
	u := fmt.Sprintf("%s/me/messages/%s?$select=id,subject,from,receivedDateTime,bodyPreview,body,isRead",
		c.graphURL, url.PathEscape(messageID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var m Message
	if err := c.doGraph(req, accessToken, proxyURL, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) doGraph(req *http.Request, accessToken, proxyURL string, out any) error {
	hc, err := httpClient(proxyURL, 30*time.Second)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrTemporarilyUnavailable, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("%w: Graph 401（access_token 失效）", ErrInvalidGrant)
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: Graph HTTP %d", ErrTemporarilyUnavailable, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Graph HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 300)]))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析 Graph 响应失败: %w", err)
	}
	return nil
}

func trimDesc(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const max = 200
	if len(s) > max {
		s = s[:max] + "..."
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
