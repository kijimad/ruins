package world

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitWorld(t *testing.T) {
	t.Parallel()
	t.Run("InitWorldが動作する", func(t *testing.T) {
		t.Parallel()
		gameComponents := &gc.Components{}

		world, err := InitWorld(gameComponents)

		require.NoError(t, err)
		assert.NotNil(t, world.World)
		assert.NotNil(t, world.Components)
		assert.NotNil(t, world.Resources)
		assert.NotNil(t, world.Components)
	})
}

func TestWorld_GetWorld(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents)
	require.NoError(t, err)

	assert.Equal(t, w.World, w.GetWorld())
}

func TestWorld_Components(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents)
	require.NoError(t, err)

	assert.Equal(t, gameComponents, w.Components)
}

func TestInitWorld_SingletonEntity(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents)
	require.NoError(t, err)

	// SingletonEntityが設定されていることを確認
	singleton := w.Resources.SingletonEntity
	assert.True(t, w.Components.GameLog.Has(singleton))
	assert.True(t, w.Components.GameProgress.Has(singleton))
	assert.True(t, w.Components.DungeonState.Has(singleton))
	assert.True(t, w.Components.TurnState.Has(singleton))
	assert.True(t, w.Components.SpatialIndex.Has(singleton))
}
