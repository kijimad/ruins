//go:build !js

package crashreport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRecord_諸元を埋める(t *testing.T) {
	t.Parallel()

	rec := buildRecord("boom", []byte("goroutine 1 [running]:"), "*states.MainMenuState")

	assert.Equal(t, "FATAL", rec.Level, "logger の Level.String() の表記に揃える")
	assert.Equal(t, "crash", rec.Category)
	assert.Equal(t, "boom", rec.Message, "recover した値を文字列で載せる")
	assert.Equal(t, "*states.MainMenuState", rec.State, "渡された最上位ステート名を載せる")
	assert.Contains(t, rec.Stack, "goroutine 1", "スタックトレースを載せる")
	assert.NotEmpty(t, rec.Timestamp)
	assert.NotEmpty(t, rec.GOOS)
}

func TestBuildRecord_stateが空でも組める(t *testing.T) {
	t.Parallel()

	rec := buildRecord("x", nil, "")

	assert.Empty(t, rec.State)
	assert.Equal(t, "x", rec.Message)
}

func TestWriteRecordTo_JSONファイルを書く(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "crash")

	path := writeRecordTo(dir, CrashRecord{Level: "FATAL", Category: "crash", Message: "boom"})

	require.NotEmpty(t, path, "保存先パスを返す")
	assert.Equal(t, dir, filepath.Dir(path))

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var got CrashRecord
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "boom", got.Message)
	assert.Equal(t, "FATAL", got.Level)
}

func TestWriteRecordTo_書けない場所なら空を返す(t *testing.T) {
	t.Parallel()
	// ファイルを dir として渡すと MkdirAll が失敗する
	f := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))

	assert.Empty(t, writeRecordTo(filepath.Join(f, "crash"), CrashRecord{}))
}

// seamMu はパッケージ変数の seam を差し替えるテストを直列化する。各テストは paralleltest を満たすため
// t.Parallel を呼ぶが、この mutex で seam の critical section は1つずつ実行される。
var seamMu sync.Mutex

// withSeams は seam をロックして現在値を退避し、テスト終了時に戻して解放する。t.Parallel の後に呼ぶ。
func withSeams(t *testing.T) {
	t.Helper()
	seamMu.Lock()
	saveOnce = sync.Once{} // 各テストを未使用の Once から始める
	origDir := userConfigDir
	origProvider := stateProvider.Load()
	t.Cleanup(func() {
		userConfigDir = origDir
		stateProvider.Store(origProvider)
		seamMu.Unlock()
	})
}

func TestSetStateProvider_登録した取り出し方をcurrentStateが引く(t *testing.T) {
	t.Parallel()
	withSeams(t)

	SetStateProvider(func() string { return "*states.MainMenuState" })

	assert.Equal(t, "*states.MainMenuState", currentState())
}

func TestWriteRecord_基底解決に失敗したら空を返す(t *testing.T) {
	t.Parallel()
	withSeams(t)

	userConfigDir = func() (string, error) { return "", os.ErrPermission }

	assert.Empty(t, writeRecord(CrashRecord{Message: "x"}), "UserConfigDir が引けなければ保存しない")
}

func TestCurrentState_providerがpanicしても空を返す(t *testing.T) {
	t.Parallel()
	withSeams(t)

	broke := func() string { panic("provider broke") }
	stateProvider.Store(&broke)

	assert.NotPanics(t, func() {
		assert.Empty(t, currentState())
	}, "provider の panic を握りつぶし元の panic を覆わない")
}

func TestGuard_保存して再panicする(t *testing.T) {
	t.Parallel()
	withSeams(t)

	dir := t.TempDir()
	userConfigDir = func() (string, error) { return dir, nil }

	assert.PanicsWithValue(t, "kaboom", func() {
		defer Guard()
		panic("kaboom")
	}, "握りつぶさず元の値で再 panic する")

	entries, err := os.ReadDir(filepath.Join(dir, "ruins", "crash"))
	require.NoError(t, err)
	assert.Len(t, entries, 1, "クラッシュ1件につきファイル1つ")
}
