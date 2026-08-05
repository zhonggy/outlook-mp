package pkg

import "testing"

func TestExtractCode(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{"labeled", "Your verification code is: 123456", "123456"},
		{"chinese", "验证码：654321 请尽快输入", "654321"},
		{"security code", "Microsoft account security code: 887766", "887766"},
		{"plain 6 digit", "您的代码是 445566，10 分钟内有效", "445566"},
		{"no code", "您好，这是一封普通邮件", ""},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ExtractCode(c.text); got != c.want {
				t.Fatalf("ExtractCode(%q) = %q, want %q", c.text, got, c.want)
			}
		})
	}
}

func TestExtractCodeExcludeEmailDigits(t *testing.T) {
	// 邮箱本地部分含 6 位数字时，不应误提取
	text := "account 123456@x.com 您好"
	if got := ExtractCode(text, "123456@x.com"); got != "" {
		t.Fatalf("应排除邮箱内数字, got %q", got)
	}
}

func TestSplitJoinTags(t *testing.T) {
	tags := SplitTags(" vip, 重要 ,,")
	if len(tags) != 2 || tags[0] != "vip" || tags[1] != "重要" {
		t.Fatalf("SplitTags 错误: %v", tags)
	}
	if JoinTags(tags) != "vip,重要" {
		t.Fatalf("JoinTags 错误: %q", JoinTags(tags))
	}
	if SplitTags("") != nil {
		t.Fatal("空串应返回 nil")
	}
}

func TestPasswordHashRoundtrip(t *testing.T) {
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword("secret123", hash) {
		t.Fatal("校验应通过")
	}
	if CheckPassword("wrong", hash) {
		t.Fatal("错误密码不应通过")
	}
}
