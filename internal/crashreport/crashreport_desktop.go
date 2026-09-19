//go:build !js

package crashreport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// userConfigDir は保存先の基底を解決する。テストが一時ディレクトリへ差し替えるためパッケージ変数にする。
var userConfigDir = os.UserConfigDir

// writeRecord は rec を UserConfigDir 下のファイルへ書き保存先を返す。失敗時は空を返す。
// 実行ファイル隣でなく UserConfigDir にするのは、出荷ビルドの実行ディレクトリが Program Files など
// 書き込み不可なことが多いため。
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
	// ファイル名の時刻はコロンを使わない。Windows はコロンを許さない。保存は saveOnce で1プロセス1回に
	// 限るので、ナノ秒まで入れれば再起動を跨いでも実質衝突しない。rec.at からファイル名と Timestamp を
	// 同一時刻で作る
	name := fmt.Sprintf("crash-%s-%09d.json", rec.at.Format("20060102-150405"), rec.at.Nanosecond())
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return ""
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(rec)
	// remove の前に閉じる。Windows は open 中のファイルを消せない。Close 時の書き込み失敗も拾う
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		// 書きかけの空ファイルを残さない。中身の無いクラッシュ記録でユーザーを混乱させないため
		_ = os.Remove(path)
		return ""
	}
	return path
}
