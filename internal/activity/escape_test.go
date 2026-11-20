package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUseItemBehavior_修理した脱出の鍵を使うと脱出を頼む(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	for id, count := range map[string]int{"broken_escape_core": 1, "signal_amplifier": 1, "power_cell": 2, "electric_wire": 4} {
		_, err := lifecycle.SpawnBackpackItem(world, id, count)
		require.NoError(t, err)
	}

	core, err := gameaction.Craft(world, "escape_core")
	require.NoError(t, err)

	b := &UseItemBehavior{}
	comp := NewUseItemActivity(core)
	require.NoError(t, b.Validate(comp, player, world))
	require.NoError(t, b.Start(comp, player, world))
	require.NoError(t, b.DoTurn(comp, player, world))
	require.NoError(t, b.Finish(comp, player, world))

	_, found := query.FindStackInInventory(world, "escape_core")
	assert.False(t, found, "使った脱出の鍵は消える")
	req := lifecycle.ConsumeStateChange(world)
	require.NotNil(t, req)
	assert.Equal(t, gc.Escape{}, req.Payload)
}
