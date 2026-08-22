package world

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestConfig は既定値で初期化したテスト用 config を返す。
func newTestConfig() *config.Config {
	c := &config.Config{Profile: config.ProfileDevelopment}
	c.ApplyProfileDefaults()
	return c
}

func TestInitWorld(t *testing.T) {
	t.Parallel()
	t.Run("InitWorldが動作する", func(t *testing.T) {
		t.Parallel()
		gameComponents := &gc.Components{}

		world, err := InitWorld(gameComponents, newTestConfig())

		require.NoError(t, err)
		assert.NotNil(t, world.ECS)
		assert.NotNil(t, world.Components)
		assert.NotNil(t, world.Resources)
		assert.NotNil(t, world.Components)
	})
}

func TestWorld_GetWorld(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents, newTestConfig())
	require.NoError(t, err)

	assert.Equal(t, w.ECS, w.GetWorld())
}

func TestWorld_Components(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents, newTestConfig())
	require.NoError(t, err)

	assert.Equal(t, gameComponents, w.Components)
}

func TestInitWorld_SingletonEntity(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents, newTestConfig())
	require.NoError(t, err)

	// SingletonEntityが設定されていることを確認
	singleton := w.Resources.SingletonEntity
	assert.True(t, w.Components.GameLog.Has(singleton))
	assert.True(t, w.Components.GameProgress.Has(singleton))
	assert.True(t, w.Components.Dungeon.Has(singleton))
	assert.True(t, w.Components.TurnState.Has(singleton))
	assert.True(t, w.Components.SpatialIndex.Has(singleton))
}

// TestWorld_ResetForNewGame は新しいゲームを始める前の後片付けを固定する。
// 前のゲームの実体が消え、シングルトンが作り直されて参照が新しい実体を指す
func TestWorld_ResetForNewGame(t *testing.T) {
	t.Parallel()
	gameComponents := &gc.Components{}
	w, err := InitWorld(gameComponents, newTestConfig())
	require.NoError(t, err)
	oldSingleton := w.Resources.SingletonEntity
	leftover := w.ECS.NewEntity()

	w.ResetForNewGame()

	assert.False(t, w.ECS.Alive(leftover), "前のゲームの実体は消える")
	assert.False(t, w.ECS.Alive(oldSingleton), "古いシングルトンも消える")
	singleton := w.Resources.SingletonEntity
	require.True(t, w.ECS.Alive(singleton), "シングルトンは作り直される")
	assert.True(t, w.Components.GameLog.Has(singleton), "シングルトンの構成が揃っている")
}
