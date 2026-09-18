//go:build !js

package crashreport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// userConfigDir は保存先の基底を解決する。既定は os.UserConfigDir。テストが一時ディレクトリへ
// 差し替えられるようパッケージ変数にする。これが唯一の保存先注入経路。
var userConfigDir = os.UserConfigDir

// writeRecord は rec を UserConfigDir 下のファイルへ書き、その保存先を返す。失敗時は空を返す。
// 実行ファイル隣でなく UserConfigDir にするのは、出荷ビルドの実行ディレクトリが Program Files など
// 書き込み不可なことが多いため。UserConfigDir はセーブ・設定と同じ親で必ず書ける。
func writeRecord(rec CrashRecord) string {
	base, err := userConfigDir()
	if err != nil {
		return ""
	}
	return writeRecordTo(filepath.Join(base, "ruins", "crash"), rec)
}

// writeRecordTo は rec を dir 下のファイルへ書き、その保存先を返す。失敗時は空を返す。
// dir を引数にとる純関数なので、テストはグローバルを触らず一時ディレクトリで検証できる。
func writeRecordTo(dir string, rec CrashRecord) string {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	// ファイル名の時刻はコロンを使わない。Windows はコロンを許さない。秒未満まで入れて同秒の衝突を避ける
	now := time.Now()
	name := fmt.Sprintf("crash-%s-%09d.json", now.Format("20060102-150405"), now.Nanosecond())
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rec); err != nil {
		// 書きかけの空ファイルを残さない。ユーザーが中身の無いクラッシュ記録を見て混乱しないため
		_ = os.Remove(path)
		return ""
	}
	return path
}
