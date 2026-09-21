package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployCube(t *testing.T) {
	t.Parallel()

	t.Run("展開で展開中マーカーが付く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
		require.NoError(t, err)

		assert.True(t, DeployCube(world, cube))
		assert.True(t, world.Components.Deployed.Has(cube), "展開中マーカーが付く")
	})

	t.Run("四方が壁で塞がれていれば展開しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
		require.NoError(t, err)
		// 隣の1マスを壁にすると全か無かで展開が拒否される
		wall := world.ECS.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 9}})
		world.Components.BlockPass.Add(wall, &gc.BlockPass{})
		query.InvalidateSpatialIndex(world)

		assert.False(t, DeployCube(world, cube))
		assert.False(t, world.Components.Deployed.Has(cube), "展開しないのでマーカーは付かない")
	})

	t.Run("斜め隣接にアイテムやpropがあれば展開しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
		require.NoError(t, err)
		// 斜め隣接(9,9)にフィールドアイテムがあっても全か無かで展開が拒否される。草の散布は密で斜めに乗る
		_, err = SpawnFieldItem(world, "wooden_sword", 9, 9, 1)
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		assert.False(t, DeployCube(world, cube))
		assert.False(t, world.Components.Deployed.Has(cube), "展開しないのでマーカーは付かない")
	})

	t.Run("既に展開中なら何もせず真を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
		require.NoError(t, err)
		require.True(t, DeployCube(world, cube))

		assert.True(t, DeployCube(world, cube))
		assert.True(t, world.Components.Deployed.Has(cube))
	})
}

func TestStowCube(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, DeployCube(world, cube))

	StowCube(world, cube)
	assert.False(t, world.Components.Deployed.Has(cube), "圧縮で展開中マーカーが外れる")
}

func TestStowCube_足元のアイテムを畳み込み展開で戻す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, DeployCube(world, cube))

	// 野営内(10,9)にフィールドアイテムを置く
	item, err := SpawnFieldItem(world, "wooden_sword", 10, 9, 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	// 圧縮で畳み込む
	StowCube(world, cube)
	require.True(t, world.Components.LocationStowed.Has(item), "野営のアイテムは LocationStowed で畳み込まれる")
	assert.Equal(t, cube, world.Components.LocationStowed.Get(item).Owner, "畳み込み先はこのキューブ")
	assert.False(t, world.Components.LocationInStorage.Has(item), "燃料タンクには入らない")
	assert.False(t, world.Components.LocationOnField.Has(item), "フィールドから外れる")
	assert.Equal(t, consts.Heat(0), query.CubeFuelTotal(world, cube), "畳み込んだ貨物は燃料に数えない")

	// 展開で戻す
	require.True(t, DeployCube(world, cube))
	assert.False(t, world.Components.LocationStowed.Has(item), "展開で LocationStowed が外れる")
	assert.True(t, world.Components.LocationOnField.Has(item), "フィールドへ戻る")
}

func TestStowCube_置いた相対位置を保持して戻す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, DeployCube(world, cube))

	// キューブから見て(0,-1)の位置に置く
	item, err := SpawnFieldItem(world, "wooden_sword", 10, 9, 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	StowCube(world, cube)

	// キューブが移動してから展開する。相対位置を保って再現する
	world.Components.GridElement.Get(cube).Coord = consts.Coord[consts.Tile]{X: 20, Y: 20}
	require.True(t, DeployCube(world, cube))

	require.True(t, world.Components.GridElement.Has(item))
	assert.Equal(t, consts.Coord[consts.Tile]{X: 20, Y: 19}, world.Components.GridElement.Get(item).Coord,
		"移動後も相対位置(0,-1)を保って戻る")
}

func TestStowCube_草などのpropは畳み込まない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	// 野営半径2内だが展開判定の8マス外(12,10)に草propを置く。展開は成立し、圧縮で吸い込まれてはいけない
	grass, err := SpawnProp(world, "grass", 12, 10)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	require.True(t, DeployCube(world, cube))
	StowCube(world, cube)

	assert.False(t, world.Components.LocationStowed.Has(grass), "草propは畳み込まない")
	assert.True(t, world.Components.LocationOnField.Has(grass), "草はフィールドに残る")
}

func TestStowDefaultCubeCargo_展開で左上に既定貨物が現れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.NoError(t, StowDefaultCubeCargo(world, cube))
	query.InvalidateSpatialIndex(world)

	require.True(t, DeployCube(world, cube))

	// 展開すると既定貨物が左上(9,9)へ現れる
	found := false
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if e == cube || !world.Components.GridElement.Has(e) {
			continue
		}
		if world.Components.GridElement.Get(e).Coord == (consts.Coord[consts.Tile]{X: 9, Y: 9}) {
			found = true
		}
	}
	assert.True(t, found, "展開で既定貨物が左上に現れる")
}

func TestStowCube_容量を超える分は足元に残す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, DeployCube(world, cube))

	a, err := SpawnFieldItem(world, "wooden_sword", 10, 9, 1)
	require.NoError(t, err)
	b, err := SpawnFieldItem(world, "wooden_sword", 11, 10, 1)
	require.NoError(t, err)
	// 容量を1つ分ちょうどに絞る。2つ目は入らず足元に残る
	world.Components.WeightCapacity.Get(cube).Max = query.GetEntityWeight(world, a)
	query.InvalidateSpatialIndex(world)

	StowCube(world, cube)

	stowed := 0
	for _, e := range []ecs.Entity{a, b} {
		if world.Components.LocationStowed.Has(e) {
			stowed++
		}
	}
	assert.Equal(t, 1, stowed, "容量に入る1つだけ畳み込み、残りは足元に残す")
}
