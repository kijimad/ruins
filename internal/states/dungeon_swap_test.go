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
	"github.com/kijimaD/ruins/internal/world/stage"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addStageEntity は指定ステージに束縛されたエンティティを1つ作る
func addStageEntity(t *testing.T, world w.World, key gc.StageKey) ecs.Entity {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.StageBound.Add(e, &gc.StageBound{Key: key})
	return e
}

// hasPortalPrev は world に上り階段プロップが存在するかを返す。
// 本番の findPortal と違い ActiveFilter を使わず退避中ステージも含めて全ステージを見る。
// 「生成されたどの階にも上り階段があるか」を確かめるテスト専用の意図
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
	key1 := dungeonStageKey(dungeon.DungeonDebug.Name, 1)
	pos1, _, err := st.spawnFloor(world, 1, def, key1)
	require.NoError(t, err)
	require.NoError(t, lifecycle.MovePlayerToPosition(world, pos1))
	d.CurrentStage = key1

	floor1 := stage.BoundEntities(world, key1)
	require.NotEmpty(t, floor1, "floor1 が生成されている")
	assert.True(t, hasPortalPrev(world), "floor1 にも上り階段(ダンジョン脱出口)がある")

	require.NoError(t, st.descend(world))
	require.Equal(t, 2, st.Depth)
	require.Equal(t, dungeonStageKey(dungeon.DungeonDebug.Name, 2), d.CurrentStage)

	// floor1 の現物が残り、すべて退避されている
	assert.Len(t, stage.BoundEntities(world, key1), len(floor1), "floor1 の現物が残る")
	for _, e := range stage.BoundEntities(world, key1) {
		assert.True(t, world.Components.Suspended.Has(e), "floor1 は退避されている")
	}
	require.NotEmpty(t, stage.BoundEntities(world, dungeonStageKey(dungeon.DungeonDebug.Name, 2)), "floor2 が生成されている")
	assert.True(t, hasPortalPrev(world), "floor2 に上り階段がある")

	handled, aerr := st.ascend(world)
	require.NoError(t, aerr)
	require.True(t, handled, "上り階段の結線をたどって上れる")
	require.Equal(t, 1, st.Depth)
	require.Equal(t, key1, d.CurrentStage)
	assert.Len(t, stage.BoundEntities(world, key1), len(floor1), "上って戻っても floor1 は同じ現物")
	for _, e := range stage.BoundEntities(world, key1) {
		assert.False(t, world.Components.Suspended.Has(e), "floor1 は再稼働されている")
	}
}

// TestEnterDungeon_遺跡へ入り上り階段が入口へ結線される は、オーバーワールドから遺跡へ入ると
// 帯が退避され遺跡1階が生成され、遺跡の上り階段が入った入口座標へ結線されること、そして
// その上り階段でオーバーワールドの入口へ正確に戻れることを実生成で検証する。
func TestEnterDungeon_遺跡へ入り上り階段が入口へ結線される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	entrancePos := consts.Coord[consts.Tile]{X: 4, Y: 4}
	player, err := lifecycle.SpawnPlayer(world, entrancePos, "Ash")
	require.NoError(t, err)

	d := query.GetDungeon(world)
	d.CurrentStage = gc.NewOverworldStage()
	// オーバーワールド帯の現物相当。遺跡に入っている間 退避されるべき
	band := addStageEntity(t, world, gc.NewOverworldStage())

	defName := dungeon.DungeonDebug.Name
	// 本番同様、オーバーワールド State は BuilderType を持たない。ゼロ値のまま遺跡生成へ
	// 流れると PlannerFunc が nil で panic するため、その回帰も兼ねる
	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name}
	require.NoError(t, st.enterDungeon(world, defName))

	// 遺跡1階が現ステージ、オーバーワールドは退避
	dungeonKey := gc.NewNamedDungeonStage(defName, 1)
	assert.Equal(t, dungeonKey, d.CurrentStage, "現ステージは遺跡1階")
	assert.Equal(t, 1, st.Depth)
	assert.True(t, world.Components.Suspended.Has(band), "オーバーワールド帯は退避される")
	assert.NotEmpty(t, stage.BoundEntities(world, dungeonKey), "遺跡1階が生成されている")

	// 遺跡の上り階段が入口(オーバーワールド, entrancePos)へ結線されている
	upStair, _, ok := findPortal(world, gc.InteractionPortalPrev)
	require.True(t, ok, "遺跡に上り階段がある")
	require.True(t, world.Components.PortalConnection.Has(upStair), "上り階段は結線を持つ")
	conn := world.Components.PortalConnection.Get(upStair)
	assert.Equal(t, gc.NewOverworldStage(), conn.Stage, "上り階段はオーバーワールドへ結線される")
	assert.Equal(t, entrancePos, conn.Coord, "入った入口座標へ結線される")

	// 上り階段で exit → オーバーワールドへ戻り、入口へ配置される
	handled, aerr := st.ascend(world)
	require.NoError(t, aerr)
	require.True(t, handled, "上り階段の結線で地上へ戻れる")
	assert.Equal(t, gc.NewOverworldStage(), d.CurrentStage, "地上へ戻る")
	assert.Equal(t, 0, st.Depth, "地上の深度は0")
	assert.False(t, world.Components.Suspended.Has(band), "オーバーワールド帯が再稼働する")
	assert.Equal(t, entrancePos, world.Components.GridElement.Get(player).Coord, "入った入口へ戻る")
}

// TestDescend_現階を退避し訪問済み階を再稼働する は共存方式の下りを検証する。
// 訪問済みの階へ降りると、現階は破棄されず退避され、行き先は再生成でなく再稼働される。
// これが「行き来しても保持」の実挙動。実生成を通らない resume 経路で orchestration を確かめる。
func TestDescend_現階を退避し訪問済み階を再稼働する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 現在は1階
	d := query.GetDungeon(world)
	d.DefinitionName = dungeon.DungeonDebug.Name
	d.CurrentStage = dungeonStageKey(dungeon.DungeonDebug.Name, 1)
	floor1 := addStageEntity(t, world, dungeonStageKey(dungeon.DungeonDebug.Name, 1))

	// 2階は訪問済みとして退避中に置く。降りると再稼働されるべき
	floor2 := addStageEntity(t, world, dungeonStageKey(dungeon.DungeonDebug.Name, 2))
	world.Components.Suspended.Add(floor2, &gc.Suspended{})

	// 実フロア相当。2階には上り階段があり、再訪でプレイヤーはそこへ配置される
	upStair := world.ECS.NewEntity()
	world.Components.GridElement.Add(upStair, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	world.Components.Interactable.Add(upStair, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionPortalPrev},
	})
	world.Components.StageBound.Add(upStair, &gc.StageBound{Key: dungeonStageKey(dungeon.DungeonDebug.Name, 2)})
	world.Components.Suspended.Add(upStair, &gc.Suspended{})

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "Ash")
	require.NoError(t, err)

	st := &DungeonState{Depth: 1}
	require.NoError(t, st.descend(world))

	// 現階は退避され現物が残る。行き先は再稼働される
	assert.True(t, world.Components.Suspended.Has(floor1), "降りた1階は退避される")
	assert.True(t, world.ECS.Alive(floor1), "1階のエンティティは破棄されず現物が残る")
	assert.False(t, world.Components.Suspended.Has(floor2), "再訪する2階は再稼働される")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(player).Coord, "再訪でプレイヤーは2階の上り階段へ")

	// 深度と現ステージが更新される
	assert.Equal(t, 2, st.Depth)
	assert.Equal(t, 2, query.GetDungeon(world).CurrentStage.Depth)
	assert.Equal(t, dungeonStageKey(dungeon.DungeonDebug.Name, 2), query.GetDungeon(world).CurrentStage)
}

// TestAscend_上り先の下り階段へ戻る は上りで訪問済み階を再稼働し、
// プレイヤーを元々降りてきた下り階段へ戻すことを検証する。
func TestAscend_上り先の下り階段へ戻る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 現在は2階
	d := query.GetDungeon(world)
	d.CurrentStage = dungeonStageKey(dungeon.DungeonDebug.Name, 2)
	floor2 := addStageEntity(t, world, dungeonStageKey(dungeon.DungeonDebug.Name, 2))

	// 戻り先。1階の下り階段の位置
	stairsPos := consts.Coord[consts.Tile]{X: 7, Y: 8}

	// 2階の上り階段。生成時に結線された戻り先(1階・下り階段位置)を持つ
	upStair := world.ECS.NewEntity()
	world.Components.GridElement.Add(upStair, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 3}})
	world.Components.Interactable.Add(upStair, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionPortalPrev},
	})
	world.Components.StageBound.Add(upStair, &gc.StageBound{Key: dungeonStageKey(dungeon.DungeonDebug.Name, 2)})
	world.Components.PortalConnection.Add(upStair, &gc.PortalConnection{Stage: dungeonStageKey(dungeon.DungeonDebug.Name, 1), Coord: stairsPos})

	// 1階は訪問済みで退避中。再稼働されることを見る
	floor1 := addStageEntity(t, world, dungeonStageKey(dungeon.DungeonDebug.Name, 1))
	world.Components.Suspended.Add(floor1, &gc.Suspended{})

	// プレイヤーは2階の適当な位置にいる
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "Ash")
	require.NoError(t, err)

	st := &DungeonState{Depth: 2}
	handled, aerr := st.ascend(world)
	require.NoError(t, aerr)
	require.True(t, handled, "結線をたどって上れる")

	// 2階退避、1階再稼働、深度1、プレイヤーは結線の戻り先へ
	assert.True(t, world.Components.Suspended.Has(floor2), "上った2階は退避される")
	assert.False(t, world.Components.Suspended.Has(floor1), "1階は再稼働される")
	assert.Equal(t, 1, st.Depth)
	assert.Equal(t, stairsPos, world.Components.GridElement.Get(player).Coord, "プレイヤーは結線した戻り先へ戻る")
}
