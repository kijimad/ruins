package states

import (
	"testing"

	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hasPortalPrev は world に上り階段プロップが存在するかを返す
func hasPortalPrev(world w.World) bool {
	found := false
	q := ecs.NewFilter1[gc.Interactable](world.ECS).Query()
	for q.Next() {
		if slices.Contains(world.Components.Interactable.Get(q.Entity()).Interactions, gc.InteractionPortalPrev) {
			found = true
		}
	}
	return found
}

// TestRoundTrip_実生成で往復し現物が復元される は実際のマップ生成を通して往復を検証する。
// 降りると現階が退避され現物が残り、上ると同じ現物が復元される。共存方式の実挙動。
func TestRoundTrip_実生成で往復し現物が復元される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "Ash")
	require.NoError(t, err)

	d := query.GetDungeon(world)
	d.DefinitionName = dungeon.DungeonDebug.Name
	def, ok := dungeon.GetDungeon(d.DefinitionName)
	require.True(t, ok)

	st := &DungeonState{Depth: 1, DefinitionName: d.DefinitionName, BuilderType: mapplanner.PlannerTypeRandom}

	// floor1 を実生成する。OnStart の生成部相当で、UI は使わない
	key1 := dungeonStageKey(1)
	pos1, err := st.spawnFloor(world, 1, def, key1)
	require.NoError(t, err)
	require.NoError(t, lifecycle.MovePlayerToPosition(world, pos1))
	d.CurrentStage = key1

	floor1 := boundEntities(world, key1)
	require.NotEmpty(t, floor1, "floor1 が生成されている")
	assert.True(t, hasPortalPrev(world), "floor1 にも上り階段(ダンジョン脱出口)がある")

	require.NoError(t, st.descend(world))
	require.Equal(t, 2, st.Depth)
	require.Equal(t, dungeonStageKey(2), d.CurrentStage)

	// floor1 の現物が残り、すべて退避されている
	assert.Len(t, boundEntities(world, key1), len(floor1), "floor1 の現物が残る")
	for _, e := range boundEntities(world, key1) {
		assert.True(t, world.Components.Suspended.Has(e), "floor1 は退避されている")
	}
	require.NotEmpty(t, boundEntities(world, dungeonStageKey(2)), "floor2 が生成されている")
	assert.True(t, hasPortalPrev(world), "floor2 に上り階段がある")

	require.NoError(t, st.ascend(world))
	require.Equal(t, 1, st.Depth)
	require.Equal(t, key1, d.CurrentStage)
	assert.Len(t, boundEntities(world, key1), len(floor1), "上って戻っても floor1 は同じ現物")
	for _, e := range boundEntities(world, key1) {
		assert.False(t, world.Components.Suspended.Has(e), "floor1 は再稼働されている")
	}
}

// TestDescend_現階を退避し訪問済み階を再稼働する は共存方式の下りを検証する。
// 訪問済みの階へ降りると、現階は破棄されず退避され、行き先は再生成でなく再稼働される。
// これが「行き来しても保持」の実挙動。実生成を通らない resume 経路で orchestration を確かめる。
func TestDescend_現階を退避し訪問済み階を再稼働する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 現在は1階
	d := query.GetDungeon(world)
	d.CurrentStage = dungeonStageKey(1)
	d.Depth = 1
	floor1 := addStageEntity(t, world, dungeonStageKey(1))

	// 2階は訪問済みとして退避中に置く。降りると再稼働されるべき
	floor2 := addStageEntity(t, world, dungeonStageKey(2))
	world.Components.Suspended.Add(floor2, &gc.Suspended{})

	st := &DungeonState{Depth: 1}
	require.NoError(t, st.descend(world))

	// 現階は退避され現物が残る。行き先は再稼働される
	assert.True(t, world.Components.Suspended.Has(floor1), "降りた1階は退避される")
	assert.True(t, world.ECS.Alive(floor1), "1階のエンティティは破棄されず現物が残る")
	assert.False(t, world.Components.Suspended.Has(floor2), "再訪する2階は再稼働される")

	// 深度と現ステージが更新される
	assert.Equal(t, 2, st.Depth)
	assert.Equal(t, 2, query.GetDungeon(world).Depth)
	assert.Equal(t, dungeonStageKey(2), query.GetDungeon(world).CurrentStage)
}

// TestAscend_上り先の下り階段へ戻る は上りで訪問済み階を再稼働し、
// プレイヤーを元々降りてきた下り階段へ戻すことを検証する。
func TestAscend_上り先の下り階段へ戻る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 現在は2階
	d := query.GetDungeon(world)
	d.CurrentStage = dungeonStageKey(2)
	d.Depth = 2
	floor2 := addStageEntity(t, world, dungeonStageKey(2))

	// 1階は訪問済みで退避中。下り階段プロップ InteractionPortalNext を持つ
	stairsPos := consts.Coord[consts.Tile]{X: 7, Y: 8}
	stairs := world.ECS.NewEntity()
	world.Components.GridElement.Add(stairs, &gc.GridElement{Coord: stairsPos})
	world.Components.Interactable.Add(stairs, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionPortalNext},
	})
	world.Components.StageBound.Add(stairs, &gc.StageBound{Key: dungeonStageKey(1)})
	world.Components.Suspended.Add(stairs, &gc.Suspended{})

	// プレイヤーは2階の適当な位置にいる
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "Ash")
	require.NoError(t, err)

	st := &DungeonState{Depth: 2}
	require.NoError(t, st.ascend(world))

	// 2階退避、1階再稼働、深度1、プレイヤーは1階の下り階段へ戻る
	assert.True(t, world.Components.Suspended.Has(floor2), "上った2階は退避される")
	assert.False(t, world.Components.Suspended.Has(stairs), "1階は再稼働される")
	assert.Equal(t, 1, st.Depth)
	assert.Equal(t, stairsPos, world.Components.GridElement.Get(player).Coord, "プレイヤーは降りてきた下り階段へ戻る")
}
