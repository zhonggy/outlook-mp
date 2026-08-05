package service

import (
	"testing"
)

func TestParseImportTextFormats(t *testing.T) {
	text := `
# 注释行应忽略
a@hotmail.com----pass1----cid1----rt1
b@outlook.com----pass2----cid2----rt2
邮箱: c@hotmail.com | 密码: pass3 | 姓名: Smith John | 生日: 1990-01-01 | 注册时间: 2026-01-01 10:00:00
d@hotmail.com,pass4,cid4,rt4
`
	items := ParseImportText(text)
	if len(items) != 4 {
		t.Fatalf("应解析 4 条, got %d: %+v", len(items), items)
	}
	if items[0].Email != "a@hotmail.com" || items[0].RefreshToken != "rt1" || items[0].ClientID != "cid1" {
		t.Fatalf("---- 格式解析错误: %+v", items[0])
	}
	if items[2].Email != "c@hotmail.com" || items[2].Password != "pass3" {
		t.Fatalf("管道格式解析错误: %+v", items[2])
	}
	if items[3].Email != "d@hotmail.com" || items[3].RefreshToken != "rt4" {
		t.Fatalf("CSV 格式解析错误: %+v", items[3])
	}
}

func TestParseImportJSON(t *testing.T) {
	data := []byte(`[
		{"email":"x@hotmail.com","password":"p","client_id":"c","refresh_token":"r"},
		{"email":"y@hotmail.com","refresh_token":"r2"}
	]`)
	items, err := ParseImportJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].RefreshToken != "r" || items[1].Email != "y@hotmail.com" {
		t.Fatalf("JSON 解析错误: %+v", items)
	}
	if _, err := ParseImportJSON([]byte("not json")); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}
