package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
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
