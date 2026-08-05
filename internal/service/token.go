package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"outlook-manager/internal/model"
	"outlook-manager/internal/msgraph"
)

// RefreshResult 单账号刷新结果。
type RefreshResult struct {
	Status  string // success / fail / skip
	Message string
}

// RefreshAccount 刷新单账号 token 并更新状态。
// 成功：回写新 refresh_token（微软轮换）与状态 healthy；invalid_grant → dead；临时错误 → error。
func (s *Service) RefreshAccount(ctx context.Context, acc *model.Account) RefreshResult {
	start := time.Now()
	if acc.RefreshToken == "" || acc.ClientID == "" {
		res := RefreshResult{Status: model.TaskSkip, Message: "缺少 refresh_token 或 client_id"}
		s.logTask(model.TaskRefresh, acc, res.Status, res.Message, start)
		return res
	}
	tr, err := s.MS.RefreshAccessToken(ctx, acc.ClientID, acc.RefreshToken, s.proxyFor(acc))
	now := time.Now()
	if err != nil {
		switch {
		case errors.Is(err, msgraph.ErrInvalidGrant):
			_ = s.Accounts.Update(acc.ID, map[string]any{
				"status":        model.StatusDead,
				"status_reason": err.Error(),
				"fail_count":    acc.FailCount + 1,
				"last_check_at": &now,
			})
			res := RefreshResult{Status: model.TaskFail, Message: "凭据失效: " + err.Error()}
			s.logTask(model.TaskRefresh, acc, res.Status, res.Message, start)
			return res
		case errors.Is(err, msgraph.ErrTemporarilyUnavailable):
			_ = s.Accounts.Update(acc.ID, map[string]any{
				"status":        model.StatusError,
				"status_reason": err.Error(),
				"last_check_at": &now,
			})
			res := RefreshResult{Status: model.TaskFail, Message: "临时故障: " + err.Error()}
			s.logTask(model.TaskRefresh, acc, res.Status, res.Message, start)
			return res
		default:
			_ = s.Accounts.Update(acc.ID, map[string]any{
				"status":        model.StatusError,
				"status_reason": err.Error(),
				"fail_count":    acc.FailCount + 1,
				"last_check_at": &now,
			})
			res := RefreshResult{Status: model.TaskFail, Message: err.Error()}
			s.logTask(model.TaskRefresh, acc, res.Status, res.Message, start)
			return res
		}
	}

	expires := now.Add(time.Duration(tr.ExpiresIn) * time.Second)
	_ = s.Accounts.Update(acc.ID, map[string]any{
		"refresh_token":    tr.RefreshToken,
		"access_token":     tr.AccessToken,
		"token_expires_at": &expires,
		"status":           model.StatusHealthy,
		"status_reason":    "",
		"fail_count":       0,
		"last_refresh_at":  &now,
		"last_check_at":    &now,
	})
	acc.RefreshToken = tr.RefreshToken
	acc.AccessToken = tr.AccessToken
	res := RefreshResult{Status: model.TaskSuccess, Message: "刷新成功"}
	s.logTask(model.TaskRefresh, acc, res.Status, res.Message, start)
	return res
}

// EnsureAccessToken 返回可用 access_token；过期/为空时先刷新。
func (s *Service) EnsureAccessToken(ctx context.Context, acc *model.Account) (string, error) {
	if acc.AccessToken != "" && acc.TokenExpiresAt != nil &&
		time.Until(*acc.TokenExpiresAt) > 5*time.Minute {
		return acc.AccessToken, nil
	}
	res := s.RefreshAccount(ctx, acc)
	if res.Status != model.TaskSuccess {
		return "", fmt.Errorf("刷新 token 失败: %s", res.Message)
	}
	return acc.AccessToken, nil
}

// CheckHealth 健康检测：刷新 token + 调 Graph /me 验证真实可用。
func (s *Service) CheckHealth(ctx context.Context, acc *model.Account) RefreshResult {
	start := time.Now()
	token, err := s.EnsureAccessToken(ctx, acc)
	if err != nil {
		res := RefreshResult{Status: model.TaskFail, Message: err.Error()}
		s.logTask(model.TaskHealth, acc, res.Status, res.Message, start)
		return res
	}
	if err := s.MS.GetMe(ctx, token, s.proxyFor(acc)); err != nil {
		now := time.Now()
		status := model.StatusError
		if errors.Is(err, msgraph.ErrInvalidGrant) {
			status = model.StatusDead
		}
		_ = s.Accounts.Update(acc.ID, map[string]any{
			"status": status, "status_reason": err.Error(), "last_check_at": &now,
		})
		res := RefreshResult{Status: model.TaskFail, Message: "Graph 探测失败: " + err.Error()}
		s.logTask(model.TaskHealth, acc, res.Status, res.Message, start)
		return res
	}
	now := time.Now()
	_ = s.Accounts.Update(acc.ID, map[string]any{
		"status": model.StatusHealthy, "status_reason": "", "last_check_at": &now,
	})
	res := RefreshResult{Status: model.TaskSuccess, Message: "健康"}
	s.logTask(model.TaskHealth, acc, res.Status, res.Message, start)
	return res
}

// Keepalive 保活：调 /me 并读一封信，维持账号活跃。
func (s *Service) Keepalive(ctx context.Context, acc *model.Account) RefreshResult {
	start := time.Now()
	if acc.Status == model.StatusDead {
		res := RefreshResult{Status: model.TaskSkip, Message: "账号已失效，跳过保活"}
		s.logTask(model.TaskKeepalive, acc, res.Status, res.Message, start)
		return res
	}
	token, err := s.EnsureAccessToken(ctx, acc)
	if err != nil {
		res := RefreshResult{Status: model.TaskFail, Message: err.Error()}
		s.logTask(model.TaskKeepalive, acc, res.Status, res.Message, start)
		return res
	}
	if err := s.MS.GetMe(ctx, token, s.proxyFor(acc)); err != nil {
		res := RefreshResult{Status: model.TaskFail, Message: "保活 /me 失败: " + err.Error()}
		s.logTask(model.TaskKeepalive, acc, res.Status, res.Message, start)
		return res
	}
	// 读一封信模拟真实活跃（失败不影响保活结论）
	_, _ = s.MS.ListMessages(ctx, token, 1, s.proxyFor(acc))
	now := time.Now()
	_ = s.Accounts.Update(acc.ID, map[string]any{"last_keepalive_at": &now})
	res := RefreshResult{Status: model.TaskSuccess, Message: "保活完成"}
	s.logTask(model.TaskKeepalive, acc, res.Status, res.Message, start)
	return res
}

// BatchResult 批量任务汇总。
type BatchResult struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Fail    int `json:"fail"`
	Skip    int `json:"skip"`
}

// RefreshAll 刷新全部（或指定状态）账号，带错峰间隔。
func (s *Service) RefreshAll(ctx context.Context, statusFilter string, stagger time.Duration) BatchResult {
	return s.runBatch(ctx, model.TaskRefresh, statusFilter, stagger, s.RefreshAccount)
}

// CheckAll 批量健康检测。
func (s *Service) CheckAll(ctx context.Context, statusFilter string, stagger time.Duration) BatchResult {
	return s.runBatch(ctx, model.TaskHealth, statusFilter, stagger, s.CheckHealth)
}

// CheckSelected 对指定 ID 的账号逐个健康检测（前端多选批量操作）。
func (s *Service) CheckSelected(ctx context.Context, ids []uint) BatchResult {
	return s.runOnIDs(ctx, ids, s.CheckHealth)
}

// RefreshSelected 对指定 ID 的账号逐个刷新 token（前端多选批量操作）。
func (s *Service) RefreshSelected(ctx context.Context, ids []uint) BatchResult {
	return s.runOnIDs(ctx, ids, s.RefreshAccount)
}

// runOnIDs 遍历 ID 列表执行任务并汇总；账号不存在计为失败。
func (s *Service) runOnIDs(ctx context.Context, ids []uint,
	fn func(context.Context, *model.Account) RefreshResult) BatchResult {
	var res BatchResult
	for _, id := range ids {
		if ctx.Err() != nil {
			break
		}
		acc, err := s.Accounts.ByID(id)
		if err != nil {
			res.Total++
			res.Fail++
			continue
		}
		out := fn(ctx, acc)
		res.Total++
		switch out.Status {
		case model.TaskSuccess:
			res.Success++
		case model.TaskSkip:
			res.Skip++
		default:
			res.Fail++
		}
	}
	return res
}

// KeepaliveAll 批量保活。
func (s *Service) KeepaliveAll(ctx context.Context, stagger time.Duration) BatchResult {
	return s.runBatch(ctx, model.TaskKeepalive, "", stagger, s.Keepalive)
}

func (s *Service) runBatch(ctx context.Context, taskType, statusFilter string, stagger time.Duration,
	fn func(context.Context, *model.Account) RefreshResult) BatchResult {
	var accounts []model.Account
	q := s.DB.Where("refresh_token <> ''")
	if statusFilter != "" {
		q = q.Where("status = ?", statusFilter)
	}
	if err := q.Order("id ASC").Find(&accounts).Error; err != nil {
		return BatchResult{}
	}
	var res BatchResult
	for i := range accounts {
		if ctx.Err() != nil {
			break
		}
		acc := accounts[i]
		out := fn(ctx, &acc)
		res.Total++
		switch out.Status {
		case model.TaskSuccess:
			res.Success++
		case model.TaskSkip:
			res.Skip++
		default:
			res.Fail++
		}
		if stagger > 0 {
			select {
			case <-ctx.Done():
				return res
			case <-time.After(stagger):
			}
		}
	}
	return res
}
