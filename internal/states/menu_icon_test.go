package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemIconCell(t *testing.T) {
	t.Parallel()

	t.Run("残量を持たない物はゲージを重ねない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sword, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 1, 1, 1)
		require.NoError(t, err)

		assert.Equal(t, styled.IconCell(menuIcon(world, sword)), itemIconCell(world, sword))
	})

	t.Run("残量を持つ物は残量の割合のゲージを重ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		torch, err := lifecycle.SpawnFieldItem(world, "torch", 1, 1, 1)
		require.NoError(t, err)
		world.Components.Charge.Get(torch).Current = 150

		want := styled.IconCell(menuIcon(world, torch))
		want.Gauge = &styled.CellGauge{Ratio: 0.25, Color: theme.HealthGaugeColor(0.25)}
		assert.Equal(t, want, itemIconCell(world, torch))
	})
}
