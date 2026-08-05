package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"outlook-manager/internal/model"
	"outlook-manager/internal/repository"
)

// ImportItem 单条导入项。
type ImportItem struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
	Tags         string `json:"tags"`
	GroupName    string `json:"group_name"`
	Remark       string `json:"remark"`
	Source       string `json:"source"`
}

// ImportResult 导入汇总。
type ImportResult struct {
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// ImportAccounts 批量 upsert 导入。
func (s *Service) ImportAccounts(items []ImportItem, defaultSource string) ImportResult {
	var res ImportResult
	for i := range items {
		it := items[i]
		it.Email = strings.TrimSpace(strings.ToLower(it.Email))
		if it.Email == "" || !strings.Contains(it.Email, "@") {
			res.Skipped++
			res.Errors = append(res.Errors, fmt.Sprintf("第 %d 条 email 非法: %q", i+1, it.Email))
			continue
		}
		src := it.Source
		if src == "" {
			src = defaultSource
		}
		acc := &model.Account{
			Email:        it.Email,
			Password:     it.Password,
			ClientID:     it.ClientID,
			RefreshToken: it.RefreshToken,
			Tags:         it.Tags,
			GroupName:    it.GroupName,
			Remark:       it.Remark,
			Source:       src,
		}
		_, created, err := s.Accounts.Upsert(acc)
		if err != nil {
			res.Skipped++
			res.Errors = append(res.Errors, fmt.Sprintf("%s: %v", it.Email, err))
			continue
		}
		if created {
			res.Created++
		} else {
			res.Updated++
		}
	}
	return res
}

// ParseImportText 解析文本导入（自动识别两种格式）：
//  1. tokens_formatted: email----password----client_id----refresh_token
//  2. accounts.txt: 邮箱: x | 密码: y | 姓名: ... | 生日: ... | 注册时间: ...
func ParseImportText(text string) []ImportItem {
	var out []ImportItem
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "----") {
			parts := strings.Split(line, "----")
			item := ImportItem{Email: strings.TrimSpace(parts[0])}
			if len(parts) > 1 {
				item.Password = strings.TrimSpace(parts[1])
			}
			if len(parts) > 2 {
				item.ClientID = strings.TrimSpace(parts[2])
			}
			if len(parts) > 3 {
				item.RefreshToken = strings.TrimSpace(parts[3])
			}
			out = append(out, item)
			continue
		}
		if strings.Contains(line, "|") && strings.Contains(line, ":") {
			kv := map[string]string{}
			for _, seg := range strings.Split(line, "|") {
				if idx := strings.Index(seg, ":"); idx > 0 {
					kv[strings.TrimSpace(seg[:idx])] = strings.TrimSpace(seg[idx+1:])
				}
			}
			if email := kv["邮箱"]; email != "" {
				out = append(out, ImportItem{
					Email:    email,
					Password: kv["密码"],
				})
			}
			continue
		}
		// 兼容 CSV 行: email,password,client_id,refresh_token
		if strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) >= 1 && strings.Contains(parts[0], "@") {
				item := ImportItem{Email: strings.TrimSpace(parts[0])}
				if len(parts) > 1 {
					item.Password = strings.TrimSpace(parts[1])
				}
				if len(parts) > 2 {
					item.ClientID = strings.TrimSpace(parts[2])
				}
				if len(parts) > 3 {
					item.RefreshToken = strings.TrimSpace(parts[3])
				}
				out = append(out, item)
			}
		}
	}
	return out
}

// ParseImportJSON 解析 JSON 数组导入（tokens.json 或 [{email,...}]）。
func ParseImportJSON(data []byte) ([]ImportItem, error) {
	var items []ImportItem
	if err := json.Unmarshal(data, &items); err == nil {
		return items, nil
	}
	// tokens.json 的字段名与 ImportItem 一致（email/refresh_token/client_id），
	// 若数组元素是更宽松的 map，再兜底解析。
	var raw []map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	for _, m := range raw {
		items = append(items, ImportItem{
			Email:        strOf(m["email"]),
			Password:     strOf(m["password"]),
			ClientID:     strOf(m["client_id"]),
			RefreshToken: strOf(m["refresh_token"]),
		})
	}
	return items, nil
}

// ExportAccounts 按格式导出账号。format: json / txt / csv。
func (s *Service) ExportAccounts(f repository.AccountFilter, format string) (string, string, error) {
	// limit <= 0 视为「全部」（安全上限 50000 防失控）
	limit := f.Size
	if limit <= 0 || limit > 50000 {
		limit = 50000
	}
	items, _, err := s.Accounts.List(repository.AccountFilter{
		Keyword: f.Keyword, Status: f.Status, Tag: f.Tag, Group: f.Group, Source: f.Source,
		Page: 1, Size: limit,
	})
	if err != nil {
		return "", "", err
	}
	switch format {
	case "txt":
		var b strings.Builder
		for _, a := range items {
			fmt.Fprintf(&b, "%s----%s----%s----%s\n", a.Email, a.Password, a.ClientID, a.RefreshToken)
		}
		return b.String(), "text/plain; charset=utf-8", nil
	case "csv":
		var b strings.Builder
		w := csv.NewWriter(&b)
		_ = w.Write([]string{"email", "password", "client_id", "refresh_token", "status", "tags", "group", "created_at"})
		for _, a := range items {
			_ = w.Write([]string{a.Email, a.Password, a.ClientID, a.RefreshToken, a.Status, a.Tags, a.GroupName, a.CreatedAt.Format("2006-01-02 15:04:05")})
		}
		w.Flush()
		return b.String(), "text/csv; charset=utf-8", nil
	default: // json
		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return "", "", err
		}
		return string(data), "application/json; charset=utf-8", nil
	}
}

func strOf(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
