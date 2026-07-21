package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewResumeStateFactory はロード復元の復帰先が保存内容で正しく分岐することを固定する。
// これがないと addLoadSlot が常に DungeonState を選び、シームレスワールドのロードが壊れる。
func TestNewResumeStateFactory_シームレスならOverworld(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// 現ステージのメタに帯データを持たせる。以後この帯データの有無がオーバーワールド判定を兼ねる
	query.EnsureSeamlessBand(world).Active = true

	state, err := newResumeStateFactory(world)()
	require.NoError(t, err)
	st, ok := state.(*DungeonState)
	require.True(t, ok, "統合後はどちらも DungeonState")
	assert.True(t, st.isSeamless(), "現ステージが帯データを持てばオーバーワールドモードで復帰する")
}

func TestNewResumeStateFactory_通常はDungeon(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// SeamlessBand.Active は既定 false

	state, err := newResumeStateFactory(world)()
	require.NoError(t, err)
	st, ok := state.(*DungeonState)
	require.True(t, ok, "通常は DungeonState で復帰する")
	assert.False(t, st.isSeamless(), "SeamlessBand 非アクティブなら通常ダンジョンモード")
}
