package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"qingye/server/internal/backup"
	"qingye/server/internal/config"
)

// runBackup 实现 `qingye backup`：把一份备份写到 w（由调用方给 os.Stdout）。
//
// db（默认）= SQLite 自身的一致性快照（VACUUM INTO），产物只有数据库；
// zip = 快照 + UPLOAD_DIR 目录下的照片日记图片，照片存在库外，只有 zip 形态
// 是「一份产物能还原全部」。
// 默认取 db 是为了对上备份工具的默认扩展名：改用 zip 必须同时告诉它 BACKUP_SQLITE_EXT=zip，
// 否则工具会拿 SQLite 魔数去校验 zip 产物，这一轮备份判失败（不静默存坏备份）。
//
// 库路径与照片路径都沿用应用自己的配置（DB_PATH / UPLOAD_DIR，外部工具不必再配一份路径
// ——应用最清楚自己的库在哪）。
//
// 日志一律走 stderr，stdout 只承载产物字节：所以这条路径不碰 gorm（logger.Default 写 stdout）、
// 不起 gin（DefaultWriter 同样是 stdout），也不执行 main 里那几步建目录的副作用。
func runBackup(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("qingye backup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr) // 用法错误与 -h 只能进 stderr，stdout 留给产物
	format := fs.String("format", "db", "导出格式：db（SQLite 一致性快照）或 zip（快照 + 照片）")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil // -h 已自行打印到 stderr，不算失败
		}
		return err
	}

	cfg := config.Load()
	if strings.TrimSpace(cfg.DBPath) == "" {
		return errors.New("数据库路径为空（DB_PATH）")
	}

	conn, err := backup.OpenForExport(cfg.DBPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "db":
		if err := backup.WriteSnapshot(conn, w); err != nil {
			return err
		}
		log.Printf("[backup] db 快照完成（不含照片：照片在 %s 下，要一次带走全部请用 --format zip）", cfg.UploadDir)
		return nil
	case "zip":
		count, err := backup.WriteZIP(conn, cfg.UploadDir, w)
		if err != nil {
			return err
		}
		log.Printf("[backup] zip 打包完成：库快照 + %d 张照片", count)
		return nil
	default:
		return fmt.Errorf("未知格式 %q（可选 db / zip）", *format)
	}
}
