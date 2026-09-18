package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewContinueState は継続起動の2契約を固定する。オートセーブがあれば最新から復帰し、
// 無ければ ok=false でフォールバックする。セーブ先は WithSaveDir で隔離する。

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
