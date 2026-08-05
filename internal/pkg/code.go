package pkg

import (
	"regexp"
	"strings"
)

// 验证码提取：优先带语义标签的码，其次独立 4-8 位数字。
var (
	labeledCodeRe = regexp.MustCompile(`(?i)(?:verification code|security code|code is|验证码|代码)[:：\s]*(\d{4,8})`)
	plainCodeRe   = regexp.MustCompile(`(?:^|[^\d])(\d{6})(?:[^\d]|$)`)
	plainCode48Re = regexp.MustCompile(`(?:^|[^\d])(\d{4,8})(?:[^\d]|$)`)
)

// ExtractCode 从邮件文本提取验证码。exclude 用于排除邮箱地址等含数字片段的干扰。
func ExtractCode(text string, exclude ...string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if m := labeledCodeRe.FindStringSubmatch(text); len(m) == 2 {
		return m[1]
	}
	// 先尝试 6 位（微软验证码通常为 6 位），再放宽 4-8 位
	for _, re := range []*regexp.Regexp{plainCodeRe, plainCode48Re} {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			code := m[1]
			if containsAny(code, exclude) {
				continue
			}
			return code
		}
	}
	return ""
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if sub != "" && strings.Contains(sub, s) {
			return true
		}
	}
	return false
}
