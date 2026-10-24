package backup

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sqlite "github.com/glebarez/sqlite"
)

// mustRawDB 用与生产同一个纯 Go 驱动开一个裸连接，供测试造库用。
func mustRawDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open(sqlite.DriverName, "file:"+path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestOpenForExport 导出打开：不迁移 schema、不建目录，缺文件直接报错。
func TestOpenForExport(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// 数据文件不存在时报错，而不是凭空建一个空库。
	if _, err := OpenForExport(dbPath); err == nil {
		t.Fatal("不存在的库应报错")
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Errorf("不应创建数据文件, stat err=%v", err)
	}

	// 只含无关表的「旧库」：导出打开不做迁移，业务表必须原样缺席。
	raw := filepath.Join(dir, "raw.db")
	rw := mustRawDB(t, raw)
	if _, err := rw.Exec("CREATE TABLE dummy(id INTEGER)"); err != nil {
		t.Fatalf("建旧表: %v", err)
	}
	exp, err := OpenForExport(raw)
	if err != nil {
		t.Fatalf("OpenForExport: %v", err)
	}
	defer exp.Close()
	var tables int
	if err := exp.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('plants','rooms','photo_diaries')`).Scan(&tables); err != nil {
		t.Fatalf("读 sqlite_master: %v", err)
	}
	if tables != 0 {
		t.Error("导出打开不应执行迁移创建业务表")
	}

	// 路径为空直接报错，不落到默认路径去。
	if _, err := OpenForExport(""); err == nil {
		t.Error("空路径应报错")
	}
}

// TestWriteSnapshot 快照：产出是合法 SQLite 文件（魔数），WAL 里已提交的数据也在。
func TestWriteSnapshot(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	rw := mustRawDB(t, dbPath)
	if _, err := rw.Exec(`PRAGMA journal_mode=WAL; CREATE TABLE plants(id INTEGER PRIMARY KEY, name TEXT); INSERT INTO plants(name) VALUES ('绿萝')`); err != nil {
		t.Fatalf("造数据: %v", err)
	}

	exp, err := OpenForExport(dbPath)
	if err != nil {
		t.Fatalf("OpenForExport: %v", err)
	}
	defer exp.Close()

	var buf bytes.Buffer
	if err := WriteSnapshot(exp, &buf); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "SQLite format 3\x00") {
		t.Fatal("快照缺少 SQLite 魔数")
	}

	// 快照可独立打开并读出数据（VACUUM INTO 产物是 delete 日志模式，无 -wal 伴随）。
	snapPath := filepath.Join(dir, "snapshot.db")
	if err := os.WriteFile(snapPath, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("写快照文件: %v", err)
	}
	snap := mustRawDB(t, snapPath)
	var name string
	if err := snap.QueryRow(`SELECT name FROM plants LIMIT 1`).Scan(&name); err != nil {
		t.Fatalf("读快照: %v", err)
	}
	if name != "绿萝" {
		t.Errorf("快照数据不符: %q", name)
	}
}

// TestWriteZIP zip：库快照在包根、uploads 文件按相对路径收进 uploads/ 前缀、
// 非常规文件跳过、上传目录不存在不算错。
func TestWriteZIP(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	rw := mustRawDB(t, dbPath)
	if _, err := rw.Exec(`CREATE TABLE plants(id INTEGER PRIMARY KEY, name TEXT); INSERT INTO plants(name) VALUES ('龟背竹')`); err != nil {
		t.Fatalf("造数据: %v", err)
	}
	exp, err := OpenForExport(dbPath)
	if err != nil {
		t.Fatalf("OpenForExport: %v", err)
	}
	defer exp.Close()

	uploads := filepath.Join(dir, "uploads")
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploads, "a.jpg"), []byte("photo-a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(uploads, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploads, "sub", "b.jpg"), []byte("photo-b"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 符号链接必须被跳过：备份包里塞一条链接，回档时就是越界写。
	if err := os.Symlink(filepath.Join(uploads, "a.jpg"), filepath.Join(uploads, "evil.jpg")); err != nil {
		t.Skipf("无法创建符号链接: %v", err)
	}

	var buf bytes.Buffer
	count, err := WriteZIP(exp, uploads, &buf)
	if err != nil {
		t.Fatalf("WriteZIP: %v", err)
	}
	if count != 2 {
		t.Errorf("打包照片数 = %d, want 2（符号链接应跳过）", count)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("读 zip: %v", err)
	}
	entries := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("打开条目 %s: %v", f.Name, err)
		}
		var data bytes.Buffer
		if _, err := data.ReadFrom(rc); err != nil {
			t.Fatalf("读条目 %s: %v", f.Name, err)
		}
		rc.Close()
		entries[f.Name] = data.Bytes()
	}
	if _, ok := entries["qingye.db"]; !ok {
		t.Errorf("包根应有 %s 快照，实际条目: %v", ZipDBEntry, keys(entries))
	}
	if !strings.HasPrefix(string(entries["qingye.db"]), "SQLite format 3\x00") {
		t.Error("包内快照缺少 SQLite 魔数")
	}
	if string(entries["uploads/a.jpg"]) != "photo-a" || string(entries["uploads/sub/b.jpg"]) != "photo-b" {
		t.Errorf("uploads 文件内容不符: %v", keys(entries))
	}
	if _, ok := entries["uploads/evil.jpg"]; ok {
		t.Error("符号链接不应被打包")
	}

	// 上传目录不存在：返回 0 而不是报错，zip 里只有库快照。
	var buf2 bytes.Buffer
	count2, err := WriteZIP(exp, filepath.Join(dir, "no-such-dir"), &buf2)
	if err != nil {
		t.Fatalf("uploads 目录不存在时不应报错: %v", err)
	}
	if count2 != 0 {
		t.Errorf("count = %d, want 0", count2)
	}
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
