package lifecycle_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

// buildShelterRoom は (10,10)-(14,14) の外周を壁にした部屋を組み、北壁 (12,10) に扉を置く。
// 屋内床 3x3 は ShelterFull、withOutdoorFloor なら扉の外 (12,9) に ShelterNone の屋外床を置く。
// 戻り値は扉と、屋内中央 (12,12) と屋外 (12,9) の床タイル。屋外床を置かないとき outdoorFloor はゼロ値
func buildShelterRoom(world w.World, doorOpen, withOutdoorFloor bool) (door, indoorFloor, outdoorFloor ecs.Entity) {
	for x := consts.Tile(10); x <= 14; x++ {
		for y := consts.Tile(10); y <= 14; y++ {
			onEdge := x == 10 || x == 14 || y == 10 || y == 14
			if x == 12 && y == 10 {
				continue // 扉の位置
			}
			e := world.ECS.NewEntity()
			world.Components.GridElement.Add(e, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
			if onEdge {
				world.Components.BlockPass.Add(e, &gc.BlockPass{})
				continue
			}
			world.Components.TileEnvironment.Add(e, &gc.TileEnvironment{Shelter: gc.ShelterFull})
			if x == 12 && y == 12 {
				indoorFloor = e
			}
		}
	}

	door = world.ECS.NewEntity()
	world.Components.GridElement.Add(door, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 12, Y: 10}})
	world.Components.Door.Add(door, &gc.Door{IsOpen: doorOpen, Orientation: gc.DoorOrientationHorizontal})
	if !doorOpen {
		world.Components.BlockPass.Add(door, &gc.BlockPass{})
	}

	if withOutdoorFloor {
		outdoorFloor = world.ECS.NewEntity()
		world.Components.GridElement.Add(outdoorFloor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 12, Y: 9}})
		world.Components.TileEnvironment.Add(outdoorFloor, &gc.TileEnvironment{Shelter: gc.ShelterNone})
	}

	return door, indoorFloor, outdoorFloor
}

func TestOpenDoor_扉を開けると屋内が屋外に繋がり冷気が入る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	door, indoorFloor, outdoorFloor := buildShelterRoom(world, false, true)

	require.NoError(t, lifecycle.OpenDoor(world, door))

	assert.Equal(t, gc.ShelterNone, world.Components.TileEnvironment.Get(indoorFloor).Shelter,
		"扉が開くと屋内の床は屋外扱いになる")
	assert.Equal(t, gc.ShelterNone, world.Components.TileEnvironment.Get(outdoorFloor).Shelter,
		"屋外の床は屋外のまま")
}

func TestCloseDoor_扉を閉じると屋内が囲われに戻る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	door, indoorFloor, outdoorFloor := buildShelterRoom(world, true, true)

	// 開いた状態の囲われへ揃えてから閉じる。屋内は冷気が入った状態から始まる
	indoor := world.Components.TileEnvironment.Get(indoorFloor)
	indoor.Shelter = gc.ShelterNone

	require.NoError(t, lifecycle.CloseDoor(world, door))

	assert.Equal(t, gc.ShelterFull, world.Components.TileEnvironment.Get(indoorFloor).Shelter,
		"扉が閉じると屋内の床は囲われに戻る")
	assert.Equal(t, gc.ShelterNone, world.Components.TileEnvironment.Get(outdoorFloor).Shelter,
		"屋外の床は屋外のまま")
}

func TestOpenDoor_再計算範囲の外へ抜けても既存の屋外タイルを見なければ書き換えない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 扉の外に屋外床が1枚も無い部屋。開くと領域は再計算範囲の外周へ達するが、
	// ShelterNone のタイルを見ないため範囲より大きい囲いと区別できず、保守則で現状維持になる
	door, indoorFloor, _ := buildShelterRoom(world, false, false)

	require.NoError(t, lifecycle.OpenDoor(world, door))

	assert.Equal(t, gc.ShelterFull, world.Components.TileEnvironment.Get(indoorFloor).Shelter,
		"屋外の証拠が無い領域は誤って冷やさず現状維持にする")
}
