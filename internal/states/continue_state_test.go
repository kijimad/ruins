package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewContinueState は継続起動の契約を固定する。手動と自動の全スロットから最新を読み、
// 読めるセーブが無ければ ok=false でフォールバックする。セーブ先は WithSaveDir で隔離する。

func TestNewContinueState_オートセーブが無ければフォールバックする(t *testing.T) {
	t.Parallel()

	saveManager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	state, ok := NewContinueState(world, saveManager)

	assert.False(t, ok, "読み込めるオートセーブが無ければ ok=false でフォールバックさせる")
	assert.Nil(t, state)
}

func TestNewContinueState_最新オートセーブから復帰する(t *testing.T) {
	t.Parallel()

	saveManager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	// 保存元と復元先は起動時と同じく別ワールドにする
	saved := testutil.InitTestWorld(t)
	require.NoError(t, saveManager.AutoSave(saved))

	fresh := testutil.InitTestWorld(t)
	state, ok := NewContinueState(fresh, saveManager)

	require.True(t, ok, "オートセーブがあれば読み込んで復帰する")
	require.NotNil(t, state)
	_, isDungeon := state.(*DungeonState)
	assert.True(t, isDungeon, "統合後の復帰先は DungeonState")
}

func TestNewContinueState_手動が最新なら手動を読む(t *testing.T) {
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
	state, ok := NewContinueState(fresh, saveManager)

	require.True(t, ok)
	require.NotNil(t, state)
	assert.Equal(t, consts.Turn(20), query.GetTurnState(fresh).TurnNumber,
		"手動が最新なら自動でなく手動を読む")
}
