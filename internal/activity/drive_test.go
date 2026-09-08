package activity

import (
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addDriveFuel はキューブ収納に燃料アイテムを1つ足す
func addDriveFuel(t *testing.T, world w.World, cube ecs.Entity, kind oapi.Material, mg consts.Milligram) {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.Material.Add(e, &gc.Material{Kind: kind})
	world.Components.Weight.Add(e, &gc.Weight{Milligram: mg})
	world.Components.LocationInStorage.Add(e, &gc.LocationInStorage{Owner: cube})
}

func TestExecuteInteraction_乗車でDrivingが付く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	_, err = ExecuteInteraction(player, cube, gc.InteractionDrive, world)
	require.NoError(t, err)

	require.True(t, world.Components.Driving.Has(player), "乗車で Driving が付く")
	assert.Equal(t, cube, world.Components.Driving.Get(player).Vehicle, "運転対象はそのキューブ")

	var logged bool
	for _, e := range query.GetGameLog(world).GetRecentEntries(10) {
		if strings.Contains(e.Text(), "board the cube") {
			logged = true
		}
	}
	assert.True(t, logged, "乗車をゲームログに出す")
}

func TestExecuteMoveAction_運転中はキューブとプレイヤーが一緒に動く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	addDriveFuel(t, world, cube, oapi.COAL, consts.Milligram(5*consts.MilligramPerKg))
	world.Components.Driving.Add(player, &gc.Driving{Vehicle: cube})

	require.NoError(t, ExecuteMoveAction(world, gc.DirectionRight))

	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord, "キューブが進む")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(player).Coord, "プレイヤーも同乗して進む")
	assert.Less(t, int(query.CubeFuelTotal(world, cube)), 4000, "燃料を消費する")
}

func TestExecuteMoveAction_プレイヤーが動けなければキューブも進まず燃料も残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	addDriveFuel(t, world, cube, oapi.COAL, consts.Milligram(5*consts.MilligramPerKg))
	world.Components.Driving.Add(player, &gc.Driving{Vehicle: cube})

	// プレイヤーを重量超過にして移動を成立させない。同乗ずれと燃料の空消費を防ぐ回帰
	require.True(t, world.Components.WeightCapacity.Has(player))
	wc := world.Components.WeightCapacity.Get(player)
	wc.Current = wc.Max*2 + 1

	require.NoError(t, ExecuteMoveAction(world, gc.DirectionRight))

	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "プレイヤーが動けないならキューブも進まない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(player).Coord, "プレイヤーも動かない")
	assert.Equal(t, consts.Heat(4000), query.CubeFuelTotal(world, cube), "移動が成立しないなら燃料は消費しない")
}

func TestExecuteMoveAction_燃料切れは立往生する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	// 燃料を積まずに乗車する
	world.Components.Driving.Add(player, &gc.Driving{Vehicle: cube})

	require.NoError(t, ExecuteMoveAction(world, gc.DirectionRight))

	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "燃料切れでキューブは動かない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(player).Coord, "プレイヤーも動かない")
}

func TestToolCandidates_隣接キューブの収納工具を含む(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 6, Y: 5})
	require.NoError(t, err)

	tool := world.ECS.NewEntity()
	world.Components.LocationInStorage.Add(tool, &gc.LocationInStorage{Owner: cube})

	assert.Contains(t, ToolCandidates(world, player), tool, "隣接キューブの収納工具が候補に入る")
}

func TestToolCandidates_離れたキューブの収納工具は含まない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 20, Y: 20})
	require.NoError(t, err)

	tool := world.ECS.NewEntity()
	world.Components.LocationInStorage.Add(tool, &gc.LocationInStorage{Owner: cube})

	assert.NotContains(t, ToolCandidates(world, player), tool, "離れたキューブの工具は候補に入らない")
}

func TestApplyDamage_運転中のプレイヤーも被弾する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	world.Components.Driving.Add(player, &gc.Driving{Vehicle: cube})

	hpBefore := world.Components.HP.Get(player).Current
	gameaction.ApplyDamage(world, player, 5, cube)

	assert.Less(t, world.Components.HP.Get(player).Current, hpBefore, "運転中でも entity は残り被弾する。反撃だけできない非対称")
}

func TestIsDrivingPlayer_運転中のプレイヤーを見分ける(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	assert.False(t, query.IsDrivingPlayer(world, player), "乗車前は運転中でない")
	world.Components.Driving.Add(player, &gc.Driving{Vehicle: cube})
	assert.True(t, query.IsDrivingPlayer(world, player), "乗車後は運転中")
	assert.False(t, query.IsDrivingPlayer(world, cube), "キューブ自身は運転中プレイヤーでない")
}
