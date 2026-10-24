package main

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sqlite "github.com/glebarez/sqlite"
)

// TestRunBackup 子命令行为：stdout 只承载产物字节，日志走 stderr（log 默认即 stderr），
// 未知格式报错，库文件缺失时直接失败且不创建文件。
func TestRunBackup(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "qingye.db")
	uploads := filepath.Join(dir, "uploads")
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open(sqlite.DriverName, "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE plants(id INTEGER PRIMARY KEY, name TEXT); INSERT INTO plants(name) VALUES ('薄荷')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploads, "1_a.jpg"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DB_PATH", dbPath)
	t.Setenv("UPLOAD_DIR", uploads)

	// db 快照：stdout 是合法 SQLite 文件。
	var out bytes.Buffer
	if err := runBackup([]string{"--format", "db"}, &out); err != nil {
		t.Fatalf("runBackup(db): %v", err)
	}
	if !strings.HasPrefix(out.String(), "SQLite format 3\x00") {
		t.Error("db 快照缺少 SQLite 魔数")
	}

	// zip：包根有快照、uploads/ 下有照片。
	out.Reset()
	if err := runBackup([]string{"--format", "zip"}, &out); err != nil {
		t.Fatalf("runBackup(zip): %v", err)
	}
	if !strings.HasPrefix(out.String(), "PK\x03\x04") {
		t.Errorf("zip 产物缺少 zip 魔数, got %.4q", out.String()[:min(4, out.Len())])
	}

	// 默认格式是 db（对上备份工具的默认扩展名，不必配 BACKUP_SQLITE_EXT）。
	out.Reset()
	if err := runBackup(nil, &out); err != nil {
		t.Fatalf("runBackup(默认): %v", err)
	}
	if !strings.HasPrefix(out.String(), "SQLite format 3\x00") {
		t.Error("默认格式应为 db 快照")
	}

	// 未知格式报错。
	if err := runBackup([]string{"--format", "tar"}, &bytes.Buffer{}); err == nil {
		t.Error("未知格式应报错")
	}

	// 库文件缺失：报错退出，且不静默创建文件。
	t.Setenv("DB_PATH", filepath.Join(dir, "missing.db"))
	if err := runBackup(nil, &bytes.Buffer{}); err == nil {
		t.Error("库文件缺失应报错")
	}
	if _, err := os.Stat(filepath.Join(dir, "missing.db")); !os.IsNotExist(err) {
		t.Error("备份路径不应创建数据文件")
	}
}
