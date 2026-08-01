package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCubeInteriorStage_定義名と一致する は、キューブ内部のステージ名が dungeon 定義の名前と
// 一致することを固定する。ずれると復帰時に resolveDungeonDefinition が名前解決に失敗して落ちる。
func TestNewCubeInteriorStage_定義名と一致する(t *testing.T) {
	t.Parallel()
	assert.Equal(t, gc.NewCubeInteriorStage().Name, dungeon.DungeonCubeInterior.Name(),
		"内部のステージ名は dungeon 定義名と一致する")
}

// TestEnterExitCube_内部へ入り元の位置へ戻る はキューブ内部への出入りを検証する。
// 入るとオーバーワールドが退避しキューブ内部が現ステージになり、出ると元のタイルへ戻る。
func TestEnterExitCube_内部へ入り元の位置へ戻る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	playerPos := consts.Coord[consts.Tile]{X: 4, Y: 5}
	player, err := lifecycle.SpawnPlayer(world, playerPos, "Ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	d := query.GetDungeon(world)
	d.CurrentStage = gc.NewOverworldStage()
	band := addStageEntity(t, world, gc.NewOverworldStage())

	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}

	// 入る: オーバーワールドを退避し、キューブ内部が現ステージになる
	require.NoError(t, st.enterCube(world, cube))
	interiorKey := gc.NewCubeInteriorStage()
	assert.Equal(t, interiorKey, d.CurrentStage, "現ステージはキューブ内部")
	assert.True(t, world.Components.Suspended.Has(band), "オーバーワールドは退避される")

	// 出口 prop の戻り先は入場時のプレイヤータイル
	exitProp, _, ok := findPortal(world, gc.InteractionExitCube)
	require.True(t, ok, "内部に出口がある")
	require.True(t, world.Components.PortalConnection.Has(exitProp))
	assert.Equal(t, playerPos, world.Components.PortalConnection.Get(exitProp).Coord, "戻り先は入場時のプレイヤータイル")

	// 内部は1階層。降り/上りの階段ポータルは無い。あると降りて panic するため
	_, _, hasNext := findPortal(world, gc.InteractionPortalNext)
	assert.False(t, hasNext, "内部に降り階段は無い")
	_, _, hasPrev := findPortal(world, gc.InteractionPortalPrev)
	assert.False(t, hasPrev, "内部に上り階段は無い")

	// 出る: オーバーワールドが再稼働し、入場した元タイルへ戻る
	require.NoError(t, st.exitCube(world))
	assert.Equal(t, gc.NewOverworldStage(), d.CurrentStage, "オーバーワールドへ戻る")
	assert.False(t, world.Components.Suspended.Has(band), "オーバーワールドが再稼働する")
	assert.Equal(t, playerPos, world.Components.GridElement.Get(player).Coord, "入場した元タイルへ戻る")
}

// TestEnterCube_内部で保存して読み込んでも落ちない はキューブ内部での serde 往復を検証する。
// 内部は定義を持たないランタイムステージなので、復帰時に遺跡定義として解決しようとして落ちる
// 不具合の回帰。復帰では世界が復元済みで定義解決は不要にする。
func TestEnterCube_内部で保存して読み込んでも落ちない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, "Ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	addStageEntity(t, world, gc.NewOverworldStage())

	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}
	require.NoError(t, st.enterCube(world, cube))
	interiorKey := gc.NewCubeInteriorStage()
	require.Equal(t, interiorKey, query.GetDungeon(world).CurrentStage)

	// 内部にいる状態で保存して別 world へ読み込む
	manager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	require.NoError(t, manager.SaveWorld(world, "cube_interior"))
	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "cube_interior"))

	assert.Equal(t, interiorKey, query.GetDungeon(newWorld).CurrentStage, "内部が現ステージのまま復元される")

	// OnStart はタイトル演出でUIリソースを参照するため、空の TextResources を用意する。
	// SplashFontFace は nil のままで良い
	newWorld.Resources.UIResources.Text = &resources.TextResources{}

	// 復帰状態を組み立てて開始する。内部は定義を持つので定義解決に成功し、通常ダンジョンと
	// 同じ経路で復帰できる。以前は定義解決に失敗して落ちていた
	resume, err := newResumeStateFactory(newWorld)()
	require.NoError(t, err)
	require.NoError(t, resume.OnStart(newWorld), "内部で読み込んでも定義解決で落ちない")
}

// TestEnterCube_内部ではオーバーワールド判定が偽 は、内部で大域マップを開くと帯が無く落ちる
// 不具合の回帰。地図の開閉は State 属性でなく現ステージの IsOnOverworld で判定するので、
// 内部では偽になり地図は開かない。
func TestEnterCube_内部ではオーバーワールド判定が偽(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world)) // 実オーバーワールド。帯とプレイヤーとキューブが揃う
	require.True(t, query.IsOnOverworld(world), "地上ではオーバーワールド判定が真")

	// スポーンされたキューブを引き当てて内部へ入る
	var cube ecs.Entity
	found := false
	cubeQuery := query.ActiveFilter1[gc.Pushable](world).Query()
	for cubeQuery.Next() {
		if !found {
			cube = cubeQuery.Entity()
			found = true
		}
	}
	require.True(t, found, "オーバーワールドにキューブがいる")

	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}
	require.NoError(t, st.enterCube(world, cube))
	assert.False(t, query.IsOnOverworld(world), "内部ではオーバーワールド判定が偽。地図は開かない")
}

// TestCubePanelState_内部の総重量を表示できる はコントロールパネルが現内部の全体情報、
// すなわち総重量を算出できることを検証する。
func TestCubePanelState_内部の総重量を表示できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, "Ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	addStageEntity(t, world, gc.NewOverworldStage())

	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}
	require.NoError(t, st.enterCube(world, cube))

	// 内部の床へ重量物を1つ置く。管制盤はこれを総重量として読む
	item := world.ECS.NewEntity()
	world.Components.Weight.Add(item, &gc.Weight{Milligram: 5 * consts.MilligramPerKg})
	world.Components.LocationOnField.Add(item, &gc.LocationOnField{})
	world.Components.StageBound.Add(item, &gc.StageBound{Key: gc.NewCubeInteriorStage()})

	// 内部で管制盤を開くと、置いた物の総重量が読める
	panel := &CubePanelState{}
	require.NoError(t, panel.OnStart(world))
	assert.Equal(t, consts.Milligram(5*consts.MilligramPerKg), panel.totalWeight, "内部に置いた物の総重量が管制盤に出る")
}

// TestOverworldMapState_キューブのチャンク位置を出す は大域地図にキューブのチャンク位置が
// マーカーとして載ることを検証する。
func TestOverworldMapState_キューブのチャンク位置を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world)) // プレイヤー近くにキューブを1体スポーンする

	st := &OverworldMapState{}
	require.NoError(t, st.OnStart(world))
	assert.NotEmpty(t, st.cubeCells, "大域地図にキューブのチャンク位置が載る")
}
