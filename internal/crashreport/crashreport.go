package crashreport

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/logger"
)

// saveOnce はクラッシュ保存を1プロセス1回に限る。複数箇所の Guard が発火してもファイルは1つに保つ。
var saveOnce sync.Once

// stateProvider は落ちた時点の最上位ステート名を取り出す関数。登録側と読み取り側が別 goroutine なので
// atomic で共有する。
var stateProvider atomic.Pointer[func() string]

// SetStateProvider は最上位ステート名の取り出し方を登録する。起動時に1度呼び、Save がクラッシュ時に引く。
func SetStateProvider(f func() string) { stateProvider.Store(&f) }

// Guard は defer で使う。panic を捕らえて1回だけ Save し、握りつぶさず再 panic する。
// recover は同一 goroutine の panic だけを捕らえるので、必ず通る関数の先頭へ置く。
func Guard() {
	if r := recover(); r != nil {
		saveOnce.Do(func() { Save(r, debug.Stack()) })
		panic(r) // 保存後に本来の落ち方へ戻す。stderr へのトレース出力と非0終了を保つ
	}
}

// Save はクラッシュ情報を1ファイルへ書く。desktop のみ、WASM は何もしない。失敗はベストエフォートで
// 握りつぶし、クラッシュ処理でさらに落ちて元の panic を覆わない。保存できたら道標の1行を出す。
func Save(recovered any, stack []byte) {
	rec := buildRecord(recovered, stack, currentState())
	if path := writeRecord(rec); path != "" {
		logger.New(logger.CategoryCrash).Error("crash report saved", "path", path)
	}
}

// buildRecord は諸元を集めて CrashRecord にする。ステート名は呼び出し側が渡す純関数。
func buildRecord(recovered any, stack []byte, state string) CrashRecord {
	return CrashRecord{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     logger.LevelFatal.String(),        // "FATAL"。logger の表記を単一出典にして取り違えを防ぐ
		Category:  string(logger.CategoryCrash),      // "crash"。同じく logger を単一出典にする
		Message:   fmt.Sprintf("%v", recovered),
		Version:   consts.AppVersion,
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		Steam:     consts.IsSteamBuild,
		State:     state,
		Stack:     string(stack),
	}
}

// currentState は登録済み provider を安全に引く。provider 自体が落ちても元の panic を覆わない。
func currentState() (name string) {
	p := stateProvider.Load()
	if p == nil {
		return ""
	}
	defer func() { _ = recover() }()
	return (*p)()
}

// CrashRecord は1回のクラッシュを表す構造化ログレコード。
// フィールド名は logger の JSON エントリ timestamp・level・category・message に揃える。
type CrashRecord struct {
	Timestamp string `json:"timestamp"`       // RFC3339
	Level     string `json:"level"`           // "FATAL"
	Category  string `json:"category"`        // "crash"
	Message   string `json:"message"`         // recover した値の文字列。全文は Stack にある
	Version   string `json:"version"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	Steam     bool   `json:"steam"`
	State     string `json:"state,omitempty"` // 最上位ステート名。無ければ空
	Stack     string `json:"stack"`
}
