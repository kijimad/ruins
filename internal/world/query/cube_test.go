package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestDriveFuelCost(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		total consts.Milligram
		want  consts.Heat
	}{
		{"空のキューブは基準燃料だけかかる", 0, consts.DriveFuelBase},
		{"総重量3kgで基準に3kgぶん加算される", consts.Milligram(3 * consts.MilligramPerKg), consts.DriveFuelBase + 3*consts.DriveFuelPerKg},
		{"総重量10kgで基準に10kgぶん加算される", consts.Milligram(10 * consts.MilligramPerKg), consts.DriveFuelBase + 10*consts.DriveFuelPerKg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, query.DriveFuelCost(tt.total))
		})
	}
}

// addCubeFuel はキューブ収納に材質と重量を持つ燃料アイテムを1つ足す。
// 生エンティティで足りる query 層のユニットテスト用。SpawnCube を使う統合テストは
// activity パッケージ側の同名ヘルパを使い、こちらとは抽象レベルが異なる。
func addCubeFuel(t *testing.T, world w.World, cube ecs.Entity, kind oapi.Material, mg consts.Milligram) {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.Material.Add(e, &gc.Material{Kind: kind})
	world.Components.Weight.Add(e, &gc.Weight{Milligram: mg})
	world.Components.LocationInStorage.Add(e, &gc.LocationInStorage{Owner: cube})
}

func TestCubeFuelTotal(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()

	// COAL は 800/kg。2kg と 1kg で 1600 + 800 = 2400。石は不燃で寄与しない
	addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(2*consts.MilligramPerKg))
	addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(1*consts.MilligramPerKg))
	addCubeFuel(t, world, cube, oapi.STONE, consts.Milligram(5*consts.MilligramPerKg))

	assert.Equal(t, consts.Heat(2400), query.CubeFuelTotal(world, cube))
}

func TestCubeWeight_空の収納は0(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()

	assert.Equal(t, consts.Milligram(0), query.CubeWeight(world, cube))
}

func TestCubeWeight_収納の物を合算する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()

	// 材質は重量に無関係。addCubeFuel は Weight と収納所属を付ける
	addCubeFuel(t, world, cube, oapi.STONE, consts.Milligram(2*consts.MilligramPerKg))
	addCubeFuel(t, world, cube, oapi.STONE, consts.Milligram(3*consts.MilligramPerKg))

	assert.Equal(t, consts.Milligram(5*consts.MilligramPerKg), query.CubeWeight(world, cube))
}

// TestDiscoveredChunks_北進ぶん絶対チャンク行をずらす は、探索済みタイルをチャンク粒度へ畳むとき
// X は有界なので列は割るだけ、Y は北進 NorthIndex ぶん絶対チャンク行を負へずらすことを固定する。
// マクロ地図フォグの座標の正確さに直結するので、NorthIndex>0 のシフト後を押さえる。
func TestDiscoveredChunks_北進ぶん絶対チャンク行をずらす(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	field := query.EnsureStageField(world, gc.NewOverworldStage())
	field.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 15, Y: 35}}: true,
		{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}:   true,
	}
	sb := &gc.SeamlessBand{ChunkW: 10, ChunkH: 10, NorthIndex: 2}

	got := query.DiscoveredChunks(world, sb)

	// col = tileX/chunkW。row = tileY/chunkH - NorthIndex。北へ進んだぶん絶対チャンク行が負へずれる
	assert.True(t, got[consts.Coord[consts.Chunk]{X: 1, Y: 1}], "(15,35)は列1・行(3-2)=1")
	assert.True(t, got[consts.Coord[consts.Chunk]{X: 0, Y: -2}], "(5,5)は列0・行(0-2)=-2")
	assert.Len(t, got, 2, "探索済みタイルを含むチャンクだけ返す")
}

func TestCubeWeight_別の収納の物は数えない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	other := world.ECS.NewEntity()

	addCubeFuel(t, world, cube, oapi.STONE, consts.Milligram(2*consts.MilligramPerKg))
	addCubeFuel(t, world, other, oapi.STONE, consts.Milligram(9*consts.MilligramPerKg))

	assert.Equal(t, consts.Milligram(2*consts.MilligramPerKg), query.CubeWeight(world, cube), "別の収納の物は除外する")
}
