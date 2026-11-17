package states

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResumeFromLatestSave は継続起動の契約を固定する。手動と自動の全スロットから最新を読み、
// 読めるセーブが無ければ ErrNoContinuePoint を返す。セーブ先は WithSaveDir で隔離する。

func TestResumeFromLatestSave_オートセーブが無ければフォールバックする(t *testing.T) {
	t.Parallel()

	saveManager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	state, err := ResumeFromLatestSave(world, saveManager)

	require.ErrorIs(t, err, ErrNoContinuePoint, "セーブが無いときは ErrNoContinuePoint を返す")
	assert.Nil(t, state, "セーブが無ければ state は nil でメニューへ退避させる")
}

func TestResumeFromLatestSave_最新オートセーブから復帰する(t *testing.T) {
	t.Parallel()

	saveManager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	// 保存元と復元先は起動時と同じく別ワールドにする
	saved := testutil.InitTestWorld(t)
	require.NoError(t, saveManager.AutoSave(saved))

	fresh := testutil.InitTestWorld(t)
	state, err := ResumeFromLatestSave(fresh, saveManager)

	require.NoError(t, err, "オートセーブがあれば読み込んで復帰する")
	require.NotNil(t, state)
	// 復帰先はオーバーワールドでも通常ダンジョンでも実体は DungeonState なので、IsOnOverworld の
	// 分岐に依らずこのアサートが復帰先を固定する
	_, isDungeon := state.(*DungeonState)
	assert.True(t, isDungeon, "復帰先は DungeonState")
}

func TestResumeFromLatestSave_手動が最新なら手動を読む(t *testing.T) {
	t.Parallel()

	saveManager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	// 先に古い自動セーブ。TurnNumber で識別する
	older := testutil.InitTestWorld(t)
	query.GetTurnState(older).TurnNumber = 10
	require.NoError(t, saveManager.AutoSave(older))

	// 後から新しい手動セーブ。time.Now が進むのでこちらが最新になる
	newer := testutil.InitTestWorld(t)
	query.GetTurnState(newer).TurnNumber = 20
	require.NoError(t, saveManager.SaveWorld(newer, "slot1"))

	fresh := testutil.InitTestWorld(t)
	state, err := ResumeFromLatestSave(fresh, saveManager)

	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, consts.Turn(20), query.GetTurnState(fresh).TurnNumber,
		"手動が最新なら自動でなく手動を読む")
}

func TestContinueOrMainMenu(t *testing.T) {
	t.Parallel()

	t.Run("読めるセーブがあれば続きから始める", func(t *testing.T) {
		t.Parallel()
		saveManager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
		require.NoError(t, err)
		require.NoError(t, saveManager.AutoSave(testutil.InitTestWorld(t)))

		_, isDungeon := ContinueOrMainMenu(testutil.InitTestWorld(t), saveManager).(*DungeonState)
		assert.True(t, isDungeon)
	})

	t.Run("セーブが無ければメインメニューから始める", func(t *testing.T) {
		t.Parallel()
		saveManager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
		require.NoError(t, err)

		menu, ok := ContinueOrMainMenu(testutil.InitTestWorld(t), saveManager).(*MainMenuState)
		require.True(t, ok)
		assert.False(t, menu.loadFailed)
	})

	t.Run("読めないセーブなら知らせを出すメインメニューから始める", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		saveManager, err := save.NewSerializationManager(save.WithSaveDir(dir))
		require.NoError(t, err)
		require.NoError(t, saveManager.AutoSave(testutil.InitTestWorld(t)))
		tamperSaves(t, dir)
		// 前提: セーブは一覧に出ていて、無い扱いでなく読み込みで失敗する
		_, err = ResumeFromLatestSave(testutil.InitTestWorld(t), saveManager)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrNoContinuePoint)

		menu, ok := ContinueOrMainMenu(testutil.InitTestWorld(t), saveManager).(*MainMenuState)
		require.True(t, ok)
		assert.True(t, menu.loadFailed)
	})

	t.Run("最新のセーブが壊れていれば古いセーブへ進まず知らせを出すメインメニューから始める", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		saveManager, err := save.NewSerializationManager(save.WithSaveDir(dir))
		require.NoError(t, err)
		require.NoError(t, saveManager.SaveWorld(testutil.InitTestWorld(t), "slot1"))
		broken := filepath.Join(dir, "slot2.json.gz")
		require.NoError(t, os.WriteFile(broken, []byte("broken"), 0o600))
		later := time.Now().Add(time.Hour)
		require.NoError(t, os.Chtimes(broken, later, later))

		menu, ok := ContinueOrMainMenu(testutil.InitTestWorld(t), saveManager).(*MainMenuState)
		require.True(t, ok)
		assert.True(t, menu.loadFailed)
	})
}

func TestMainMenuState_Update_読めなかった知らせを最初に一度だけ重ねる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	state, err := NewMainMenuAfterLoadFailure()
	require.NoError(t, err)
	st, ok := state.(*MainMenuState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	tr, err := st.Update(world)
	require.NoError(t, err)
	require.Equal(t, es.TransPush, tr.Type)
	require.Len(t, tr.NewStateFuncs, 1)
	pushed, err := tr.NewStateFuncs[0]()
	require.NoError(t, err)
	assert.IsType(t, &MessageState{}, pushed)

	tr, err = st.Update(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, tr.Type, "知らせは一度だけ")
}

func TestLoadSlotChoice_読めないセーブは知らせを出すメインメニューへ置き換える(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	saveManager, err := save.NewSerializationManager(save.WithSaveDir(dir))
	require.NoError(t, err)
	require.NoError(t, saveManager.SaveWorld(testutil.InitTestWorld(t), "slot1"))
	tamperSaves(t, dir)

	world := testutil.InitTestWorld(t)
	tr, err := loadSlotChoice(world, saveManager, "slot1").Run(world)
	require.NoError(t, err)
	require.Equal(t, es.TransReplace, tr.Type)
	require.Len(t, tr.NewStateFuncs, 1)
	state, err := tr.NewStateFuncs[0]()
	require.NoError(t, err)
	menu, ok := state.(*MainMenuState)
	require.True(t, ok)
	assert.True(t, menu.loadFailed)
}

// tamperSaves は dir の全てのセーブの版を書き換え、チェックサムを合わなくして読めなくする
func tamperSaves(t *testing.T, dir string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.json.gz"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, p := range paths {
		f, err := os.Open(p)
		require.NoError(t, err)
		zr, err := gzip.NewReader(f)
		require.NoError(t, err)
		var env map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(zr).Decode(&env))
		require.NoError(t, f.Close())
		env["version"], err = json.Marshal("0.0.0")
		require.NoError(t, err)

		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		require.NoError(t, json.NewEncoder(zw).Encode(env))
		require.NoError(t, zw.Close())
		require.NoError(t, os.WriteFile(p, buf.Bytes(), 0o600))
	}
}
