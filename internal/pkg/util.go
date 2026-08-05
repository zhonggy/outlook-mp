package pkg

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// RandomKey 生成 n 字节随机十六进制串（API Key / JWT secret 用）。
func RandomKey(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

// 去除易混淆字符（0/O、1/l/I）的密码字符集。
const passwordChars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"

// RandomPassword 生成 n 位随机强密码（首次启动管理员密码用）。
func RandomPassword(n int) string {
	if n < 8 {
		n = 8
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = passwordChars[int(b)%len(passwordChars)]
	}
	return string(out)
}

// SplitTags 把逗号分隔标签字符串切成去空白切片。
func SplitTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// JoinTags 把标签切片合成逗号分隔字符串。
func JoinTags(tags []string) string {
	return strings.Join(tags, ",")
}
