package world

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestInitWorld(t *testing.T) {
	t.Parallel()
	t.Run("InitWorldが動作する", func(t *testing.T) {
		t.Parallel()
		gameComponents := &gc.Components{}

		world, err := InitWorld(gameComponents)

		assert.NoError(t, err)
		assert.NotNil(t, world.Manager)
		assert.NotNil(t, world.Components)
		assert.NotNil(t, world.Resources)
		assert.NotNil(t, world.Components)
	})
}

func TestWorld_GetManager(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents)
	assert.NoError(t, err)

	assert.Equal(t, w.Manager, w.GetManager())
}

func TestWorld_GetComponents(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents)
	assert.NoError(t, err)

	assert.Equal(t, w.Components, w.GetComponents())
}

func TestInitWorld_SingletonEntity(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents)
	assert.NoError(t, err)

	// SingletonEntityが設定されていることを確認
	singleton := w.Resources.SingletonEntity
	assert.True(t, singleton.HasComponent(w.Components.GameLog))
	assert.True(t, singleton.HasComponent(w.Components.GameProgress))
	assert.True(t, singleton.HasComponent(w.Components.DungeonState))
	assert.True(t, singleton.HasComponent(w.Components.TurnState))
	assert.True(t, singleton.HasComponent(w.Components.SpatialIndex))
}
