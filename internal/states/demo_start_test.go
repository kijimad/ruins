package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemoStartState_OnStart(t *testing.T) {
	t.Parallel()

	state := NewDemoStartState()
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)

	// プレイヤーが生成されていることを確認
	_, err = query.GetPlayerEntity(world)
	assert.NoError(t, err, "プレイヤーが生成されている")
}

func TestDemoStartState_Update(t *testing.T) {
	t.Parallel()

	state := NewDemoStartState()
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	transition, err := state.Update(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransReplace, transition.Type, "TownStateへTransReplace")
}

func TestDemoStartState_Update_AfterConsumed(t *testing.T) {
	t.Parallel()

	state := NewDemoStartState()
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))

	// 1回目で遷移を消費
	_, err := state.Update(world)
	require.NoError(t, err)

	// 2回目はTransNone
	transition, err := state.Update(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, transition.Type, "消費済みの場合はTransNone")
}
