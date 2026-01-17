package world

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	gr "github.com/kijimaD/ruins/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestInitGeneric(t *testing.T) {
	t.Parallel()
	t.Run("型安全なInitGenericが動作する", func(t *testing.T) {
		t.Parallel()
		gameComponents := &gc.Components{}
		gameResources := &gr.Resources{}

		world, err := InitGeneric(gameComponents, gameResources)

		assert.NoError(t, err)
		assert.NotNil(t, world.Manager)
		assert.NotNil(t, world.Components)
		assert.NotNil(t, world.Resources)

		// 型安全性の確認
		assert.IsType(t, &gc.Components{}, world.Components.Game)
		assert.IsType(t, &gr.Resources{}, world.Resources.Game)
	})

	t.Run("型安全性が保たれている", func(t *testing.T) {
		t.Parallel()
		gameComponents := &gc.Components{}
		gameResources := &gr.Resources{}

		world, err := InitGeneric(gameComponents, gameResources)

		assert.NoError(t, err)
		// 型アサーションが不要で、直接アクセスできる
		assert.NotNil(t, world.Components.Game.Position)
		assert.NotNil(t, world.Resources.Game.ScreenDimensions)
	})
}
