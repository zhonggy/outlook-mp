package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"outlook-manager/internal/model"
	"outlook-manager/internal/msgraph"
	"outlook-manager/internal/pkg"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// FetchMails 拉取某账号的收件箱 + 垃圾邮件文件夹（不区分，合并入库），幂等 upsert。
// 返回新增数、收件箱拉取数、垃圾邮件拉取数。垃圾文件夹不存在/失败时仅记录告警，不影响收件箱。
func (s *Service) FetchMails(ctx context.Context, acc *model.Account, top int) (int, int, int, error) {
	start := time.Now()
	token, err := s.EnsureAccessToken(ctx, acc)
	if err != nil {
		s.logTask(model.TaskMail, acc, model.TaskFail, "获取 token 失败: "+err.Error(), start)
		return 0, 0, 0, err
	}
	proxy := s.proxyFor(acc)
	inbox, err := s.MS.ListMessagesInFolder(ctx, token, "inbox", top, proxy)
	if err != nil {
		s.logTask(model.TaskMail, acc, model.TaskFail, "拉信失败: "+err.Error(), start)
		return 0, 0, 0, err
	}
	junk, jerr := s.MS.ListMessagesInFolder(ctx, token, "junkemail", top, proxy)
	if jerr != nil {
		s.Log.Warn("垃圾邮件文件夹拉取失败，本次仅入库收件箱", "account", acc.Email, "err", jerr)
	}

	newCount := 0
	for _, batch := range []struct {
		msgs   []msgraph.Message
		folder string
	}{{inbox, "inbox"}, {junk, "junk"}} {
		for i := range batch.msgs {
			m := batch.msgs[i]
			preview := m.BodyPreview
			code := pkg.ExtractCode(preview+"\n"+m.Subject, acc.Email)
			rec := &model.MailMessage{
				AccountID:   acc.ID,
				MessageID:   m.ID,
				Subject:     m.Subject,
				FromAddr:    m.From.EmailAddress.Address,
				BodyPreview: preview,
				ReceivedAt:  m.ReceivedDateTime,
				IsRead:      m.IsRead,
				Folder:      batch.folder,
				Code:        code,
			}
			created, err := s.Mails.UpsertMessage(rec)
			if err != nil {
				s.Log.Warn("邮件入库失败", "account", acc.Email, "err", err)
				continue
			}
			if created {
				newCount++
			}
		}
	}
	now := time.Now()
	_ = s.Accounts.Update(acc.ID, map[string]any{"last_mail_at": &now})
	s.logTask(model.TaskMail, acc, model.TaskSuccess,
		fmt.Sprintf("收件箱 %d 封 + 垃圾 %d 封，新增 %d 封", len(inbox), len(junk), newCount), start)
	return newCount, len(inbox), len(junk), nil
}

// FetchMailBody 按需拉单封邮件全文并提取验证码。
func (s *Service) FetchMailBody(ctx context.Context, mailID uint) (*model.MailMessage, error) {
	m, err := s.Mails.ByID(mailID)
	if err != nil {
		return nil, err
	}
	if m.Body != "" {
		return m, nil
	}
	acc, err := s.Accounts.ByID(m.AccountID)
	if err != nil {
		return nil, err
	}
	token, err := s.EnsureAccessToken(ctx, acc)
	if err != nil {
		return nil, err
	}
	full, err := s.MS.GetMessage(ctx, token, m.MessageID, s.proxyFor(acc))
	if err != nil {
		return nil, err
	}
	body := full.Body.Content
	plain := strings.TrimSpace(htmlTagRe.ReplaceAllString(body, " "))
	code := m.Code
	if code == "" {
		code = pkg.ExtractCode(plain, acc.Email)
	}
	_ = s.DB.Model(m).Updates(map[string]any{"body": body, "code": code})
	m.Body = body
	m.Code = code
	return m, nil
}

// FetchAllMails 批量收信（调度用），仅处理非 dead 账号。
func (s *Service) FetchAllMails(ctx context.Context, top int, stagger time.Duration) BatchResult {
	var accounts []model.Account
	if err := s.DB.Where("refresh_token <> '' AND status <> ?", model.StatusDead).
		Order("id ASC").Find(&accounts).Error; err != nil {
		return BatchResult{}
	}
	var res BatchResult
	for i := range accounts {
		if ctx.Err() != nil {
			break
		}
		acc := accounts[i]
		_, _, _, err := s.FetchMails(ctx, &acc, top)
		res.Total++
		if err != nil {
			res.Fail++
		} else {
			res.Success++
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
