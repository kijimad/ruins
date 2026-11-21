package states

import (
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChargeColor_割合で色を選ぶ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		stage int
		want  color.RGBA
	}{
		{stage: 5, want: theme.ChargeHigh},
		{stage: 4, want: theme.ChargeHigh},
		{stage: 3, want: theme.ChargeMid},
		{stage: 2, want: theme.ChargeMid},
		{stage: 1, want: theme.ChargeLow},
		{stage: 0, want: theme.ChargeLow},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, chargeColor(tt.stage), tt.stage)
	}
}

func TestMenuIcon_残量を持たない物は縮小画像をそのまま使う(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	sword, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 1, 1, 1)
	require.NoError(t, err)

	assert.Same(t, world.Resources.Sprites.Sized(world.Components.SpriteRender.Get(sword), theme.MenuIconW), menuIcon(world, sword))
}
