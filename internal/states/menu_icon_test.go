package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMenuIcon_残量を持たない物は縮小画像をそのまま使う(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	sword, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 1, 1, 1)
	require.NoError(t, err)

	assert.Same(t, world.Resources.Sprites.Sized(world.Components.SpriteRender.Get(sword), theme.MenuIconW), menuIcon(world, sword))
}

func TestMenuIcon_残量を持つ物は段階のゲージを焼いた画像を使う(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	torch, err := lifecycle.SpawnFieldItem(world, "torch", 1, 1, 1)
	require.NoError(t, err)
	world.Components.Charge.Get(torch).Current = 300

	gauge := resources.Gauge{Filled: 3, Total: query.ChargeStageCount, Fill: theme.ChargeColor(3, query.ChargeStageCount), Empty: theme.ChargeEmpty}
	assert.Same(t, world.Resources.Sprites.SizedGauge(world.Components.SpriteRender.Get(torch), theme.MenuIconW, gauge), menuIcon(world, torch))
}
