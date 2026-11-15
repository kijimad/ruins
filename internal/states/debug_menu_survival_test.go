package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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

func TestDebugMenu_デバッグのダンジョンでメニューの全ての間取りを計画できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	raws := world.Resources.RawMaster
	stage, ok := dungeon.GetStageDefinition(raws, dungeon.DebugID)
	require.True(t, ok)
	def, ok := stage.(*dungeon.DungeonDefinition)
	require.True(t, ok)
	query.SetDungeon(world, &gc.Dungeon{CurrentStage: gc.NewDungeonStage(dungeon.DebugID, 1)})

	for _, pt := range mapplanner.ProceduralPlannerTypes(raws) {
		pt.Ground = def.Ground()
		pt.Danger = 1
		_, err := mapplanner.Plan(world, consts.MapTileWidth, consts.MapTileHeight, 1, pt)
		require.NoError(t, err, pt.Name)
	}
}
