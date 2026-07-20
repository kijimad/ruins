package states

import (
	"testing"

	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFadeTransitionState_暗転点でonBlackを実行し明転後popする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Config.DisableAnimation = false

	fired := 0
	factory := NewFadeTransitionState(func(_ w.World) error { fired++; return nil })
	state, err := factory()
	require.NoError(t, err)
	require.NoError(t, state.OnStart(world))
	fs, ok := state.(*FadeTransitionState)
	require.True(t, ok)

	// 暗転しきる前は onBlack を実行せず TransNone。fadeMs=250, deltaMs≈16.67
	for range 14 {
		trans, err := fs.Update(world)
		require.NoError(t, err)
		require.Equal(t, es.TransNone, trans.Type)
	}
	assert.Equal(t, 0, fired, "暗転しきる前は onBlack を実行しない")

	// 暗転しきると onBlack を1回実行する
	for range 5 {
		_, err := fs.Update(world)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, fired, "暗転点で onBlack を1回実行する")

	// 明転しきると自身を pop する
	popped := false
	for range 30 {
		trans, err := fs.Update(world)
		require.NoError(t, err)
		if trans.Type == es.TransPop {
			popped = true
			break
		}
	}
	assert.True(t, popped, "明転後に pop する")
	assert.Equal(t, 1, fired, "onBlack は高々1回")
}

func TestFadeTransitionState_アニメ無効時は即実行して即pop(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Config.DisableAnimation = true

	fired := 0
	factory := NewFadeTransitionState(func(_ w.World) error { fired++; return nil })
	state, err := factory()
	require.NoError(t, err)
	require.NoError(t, state.OnStart(world))
	fs, ok := state.(*FadeTransitionState)
	require.True(t, ok)

	trans, err := fs.Update(world)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, trans.Type, "アニメ無効時は即 pop する")
	assert.Equal(t, 1, fired, "onBlack を実行する")
}
