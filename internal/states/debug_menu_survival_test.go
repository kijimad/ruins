package states

import (
	"math/rand/v2"
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

func TestDebugTown_危険度に依らず敵もアイテムも置かない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	raws := world.Resources.RawMaster
	stage, ok := dungeon.GetStageDefinition(raws, dungeon.DebugTownID)
	require.True(t, ok)
	def, ok := stage.(*dungeon.DungeonDefinition)
	require.True(t, ok)
	query.SetDungeon(world, &gc.Dungeon{CurrentStage: gc.NewDungeonStage(dungeon.DebugTownID, 1)})

	pt, err := def.SelectPlanner(rand.New(rand.NewPCG(1, 0)))
	require.NoError(t, err)
	pt.EnemyTableName, pt.ItemTableName = def.EnemyTableName(), def.ItemTableName()
	pt.EnemyCount, pt.ItemCount = def.EnemyCount(), def.ItemCount()
	pt.Danger = 20
	plan, err := mapplanner.Plan(world, 0, 0, 1, pt)
	require.NoError(t, err)
	names := make([]string, 0, len(plan.NPCs))
	for _, n := range plan.NPCs {
		names = append(names, n.Name)
	}
	assert.ElementsMatch(t, []string{"merchant", "old_soldier"}, names, "絵の住人だけで敵は湧かない")
	assert.Empty(t, plan.Items, "床のアイテムを置かない")
}

func TestDebugInflictConditions_部位を書いた不調はその部位に怪我は胴に病気と寒さは全身に置く(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	require.NoError(t, debugInflictConditions(world))

	got := map[gc.ConditionType]gc.BodyPart{}
	hs := world.Components.HealthStatus.Get(player)
	for i := range hs.Parts {
		for _, c := range hs.Parts[i].Conditions {
			got[c.Type] = gc.BodyPart(i)
		}
	}
	assert.Equal(t, map[gc.ConditionType]gc.BodyPart{
		"fracture":       gc.BodyPartArms,
		"laceration":     gc.BodyPartTorso,
		"hypothermia":    gc.BodyPartWholeBody,
		"food_poisoning": gc.BodyPartWholeBody,
		"liver_illness":  gc.BodyPartWholeBody,
		"venom":          gc.BodyPartWholeBody,
		"daze":           gc.BodyPartHead,
	}, got)
}
