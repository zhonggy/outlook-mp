package service

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"outlook-manager/internal/config"
	"outlook-manager/internal/model"
	"outlook-manager/internal/repository"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := repository.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return New(db, cfg, log)
}

func TestImportAccountsUpsertAndExport(t *testing.T) {
	svc := newTestService(t)

	res := svc.ImportAccounts([]ImportItem{
		{Email: "A@Hotmail.com", Password: "p1", ClientID: "c1", RefreshToken: "rt1"},
		{Email: "b@hotmail.com", Password: "p2", ClientID: "c2", RefreshToken: "rt2"},
		{Email: "bad-email"},
	}, model.SourceImport)
	if res.Created != 2 || res.Updated != 0 || res.Skipped != 1 {
		t.Fatalf("首次导入统计错误: %+v", res)
	}

	// 重复导入（大写同邮箱）→ 更新而非新建
	res2 := svc.ImportAccounts([]ImportItem{
		{Email: "a@hotmail.com", RefreshToken: "rt1-new"},
	}, model.SourceImport)
	if res2.Updated != 1 || res2.Created != 0 {
		t.Fatalf("重复导入应更新: %+v", res2)
	}
	acc, err := svc.Accounts.ByEmail("a@hotmail.com")
	if err != nil {
		t.Fatal(err)
	}
	if acc.RefreshToken != "rt1-new" || acc.Password != "p1" {
		t.Fatalf("upsert 应更新 token 保留密码: %+v", acc)
	}

	// txt 导出格式与上游注册器导出格式兼容（按 id DESC，最新在前）
	content, _, err := svc.ExportAccounts(repository.AccountFilter{}, "txt")
	if err != nil {
		t.Fatal(err)
	}
	want := "b@hotmail.com----p2----c2----rt2\na@hotmail.com----p1----c1----rt1-new\n"
	if content != want {
		t.Fatalf("txt 导出错误:\ngot  %q\nwant %q", content, want)
	}
}

func TestStatusCountsAndGroups(t *testing.T) {
	svc := newTestService(t)
	svc.ImportAccounts([]ImportItem{
		{Email: "g1@hotmail.com", GroupName: "A组", RefreshToken: "r"},
		{Email: "g2@hotmail.com", GroupName: "A组", RefreshToken: "r"},
		{Email: "g3@hotmail.com", GroupName: "B组", RefreshToken: "r"},
	}, model.SourceImport)

	counts, err := svc.Accounts.StatusCounts()
	if err != nil {
		t.Fatal(err)
	}
	if counts[model.StatusUnknown] != 3 {
		t.Fatalf("状态计数错误: %v", counts)
	}
	groups, err := svc.Accounts.DistinctGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("分组应去重为 2 个: %v", groups)
	}
}
