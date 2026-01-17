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
