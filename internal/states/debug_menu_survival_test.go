package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDebugToggleInvincible_無敵を切り替える(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	require.NoError(t, debugToggleInvincible(world))
	assert.True(t, world.Resources.Config.DebugInvincible)
	require.NoError(t, debugToggleInvincible(world))
	assert.False(t, world.Resources.Config.DebugInvincible)
}

func TestDebugKillNearbyEnemies_近くの敵だけを倒す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	near, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10 + debugKillNearbyEnemiesRange, Y: 10}, "bat")
	require.NoError(t, err)
	far, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11 + debugKillNearbyEnemiesRange, Y: 10}, "bat")
	require.NoError(t, err)

	require.NoError(t, debugKillNearbyEnemies(world))
	assert.True(t, world.Components.Dead.Has(near))
	assert.False(t, world.Components.Dead.Has(far))
}

func TestDebugMenu_デバッグの町の間取りとダンジョンをrawから引ける(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster

	pt, ok := mapplanner.PlannerTypeByName(raws, debugTownLayout)
	require.True(t, ok, "メニューが指す間取りの行がある")
	assert.Equal(t, debugTownLayout, pt.Name)
	_, ok = dungeon.GetStageDefinition(raws, dungeon.DebugTownID)
	assert.True(t, ok, "メニューが指すダンジョンの行がある")
}
