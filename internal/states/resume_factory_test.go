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
	query.GetDungeon(world).SeamlessBand.Active = true

	state, err := newResumeStateFactory(world)()
	require.NoError(t, err)
	_, ok := state.(*OverworldState)
	assert.True(t, ok, "SeamlessBand.Active なら OverworldState で復帰する")
}

func TestNewResumeStateFactory_通常はDungeon(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// SeamlessBand.Active は既定 false

	state, err := newResumeStateFactory(world)()
	require.NoError(t, err)
	_, ok := state.(*DungeonState)
	assert.True(t, ok, "通常は DungeonState で復帰する")
}
