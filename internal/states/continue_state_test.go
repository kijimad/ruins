package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewContinueState は起動時の継続読み込みの2つの契約を固定する。make run の既定の
// 起動口なので、オートセーブがあれば最新から復帰し、無ければプロセスを落とさず ok=false で
// 新規開始へフォールバックする挙動が要になる。セーブ先は WithSaveDir で隔離して並列に回す。

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

	// オートセーブを1件作る。保存元と復元先は起動時と同じく別ワールドにする
	saved := testutil.InitTestWorld(t)
	require.NoError(t, saveManager.AutoSave(saved))

	fresh := testutil.InitTestWorld(t)
	state, ok := NewContinueState(fresh, saveManager)

	require.True(t, ok, "オートセーブがあれば読み込んで復帰する")
	require.NotNil(t, state)
	_, isDungeon := state.(*DungeonState)
	assert.True(t, isDungeon, "統合後の復帰先は DungeonState")
}
