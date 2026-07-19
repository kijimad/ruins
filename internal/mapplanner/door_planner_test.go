package mapplanner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestMetaPlanForDoor はドアテスト用の10x10のMetaPlanを生成する
func newTestMetaPlanForDoor() *MetaPlan {
	const width, height = 10, 10
	tiles := make([]oapi.Tile, width*height)
	for i := range tiles {
		tiles[i] = oapi.Tile{Name: "wall", BlockPass: true}
	}

	return &MetaPlan{
		Level: gc.Level{
			TileWidth:  consts.Tile(width),
			TileHeight: consts.Tile(height),
		},
		Tiles:     tiles,
		Rooms:     []gc.Rect{},
		Doors:     []DoorSpec{},
		RNG:       rand.New(rand.NewPCG(42, 43)),
		RawMaster: CreateTestRawMaster(),
	}
}

// setFloorRect はMetaPlan上の矩形範囲を床に設定する
func setFloorRect(mp *MetaPlan, x1, y1, x2, y2 int) {
	w := int(mp.Level.TileWidth)
	for x := x1; x <= x2; x++ {
		for y := y1; y <= y2; y++ {
			mp.Tiles[y*w+x] = oapi.Tile{Name: "floor", BlockPass: false}
		}
	}
}

func TestDoorPlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	t.Run("左右が壁のパターンでドアが配置される", func(t *testing.T) {
		t.Parallel()
		mp := newTestMetaPlanForDoor()

		// 部屋(2,3)-(5,5)を配置し、上から1タイル幅の廊下(x=3, y=1〜2)を接続
		// (3,2)がドア候補: 左右が壁、上(3,1)が床、下(3,3)が部屋の床
		setFloorRect(mp, 2, 3, 5, 5)
		mp.Tiles[1*10+3] = oapi.Tile{Name: "floor", BlockPass: false}
		mp.Tiles[2*10+3] = oapi.Tile{Name: "floor", BlockPass: false}

		planner := DoorPlanner{DoorChance: 1.0}
		require.NoError(t, planner.PlanMeta(mp))

		require.Len(t, mp.Doors, 1)
		assert.Equal(t, 3, int(mp.Doors[0].X))
		assert.Equal(t, 2, int(mp.Doors[0].Y))
	})

	t.Run("上下が壁のパターンでドアが配置される", func(t *testing.T) {
		t.Parallel()
		mp := newTestMetaPlanForDoor()

		// 部屋(3,2)-(6,6)を配置し、左から1タイル幅の廊下(x=1〜2, y=3)を接続
		// (2,3)がドア候補: 上下が壁、左(1,3)が床、右(3,3)が部屋の床
		setFloorRect(mp, 3, 2, 6, 6)
		mp.Tiles[3*10+1] = oapi.Tile{Name: "floor", BlockPass: false}
		mp.Tiles[3*10+2] = oapi.Tile{Name: "floor", BlockPass: false}

		planner := DoorPlanner{DoorChance: 1.0}
		require.NoError(t, planner.PlanMeta(mp))

		require.Len(t, mp.Doors, 1)
		assert.Equal(t, 2, int(mp.Doors[0].X))
		assert.Equal(t, 3, int(mp.Doors[0].Y))
	})

	t.Run("DoorChance=0の場合はドアが配置されない", func(t *testing.T) {
		t.Parallel()
		mp := newTestMetaPlanForDoor()

		setFloorRect(mp, 2, 3, 5, 5)
		mp.Tiles[1*10+3] = oapi.Tile{Name: "floor", BlockPass: false}
		mp.Tiles[2*10+3] = oapi.Tile{Name: "floor", BlockPass: false}

		planner := DoorPlanner{DoorChance: 0.0}
		require.NoError(t, planner.PlanMeta(mp))

		assert.Empty(t, mp.Doors)
	})

	t.Run("両側に壁がないパターンではドアが配置されない", func(t *testing.T) {
		t.Parallel()
		mp := newTestMetaPlanForDoor()

		// 部屋(2,2)-(5,5)と幅広廊下(x=1〜6, y=1)
		setFloorRect(mp, 2, 2, 5, 5)
		for x := 1; x <= 6; x++ {
			mp.Tiles[1*10+x] = oapi.Tile{Name: "floor", BlockPass: false}
		}

		planner := DoorPlanner{DoorChance: 1.0}
		require.NoError(t, planner.PlanMeta(mp))

		// 全ドアがパターンを満たしていることを確認
		for _, door := range mp.Doors {
			w := int(mp.Level.TileWidth)
			idx := int(door.Y)*w + int(door.X)
			hasLR := mp.Tiles[idx-1].BlockPass && mp.Tiles[idx+1].BlockPass
			hasTB := mp.Tiles[idx-w].BlockPass && mp.Tiles[idx+w].BlockPass
			assert.True(t, hasLR || hasTB,
				"ドア(%d,%d)の両側に壁がない", door.X, door.Y)
		}
	})

	t.Run("SmallRoomPlannerでドアが生成される", func(t *testing.T) {
		t.Parallel()

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)

		chain.PlanData.RawMaster = CreateTestRawMaster()
		require.NoError(t, chain.Plan())

		if len(chain.PlanData.Rooms) >= 2 {
			assert.NotEmpty(t, chain.PlanData.Doors, "部屋が2つ以上ある場合、廊下入口にドアが生成されるべき")
		}
	})
}

func TestIsDoorPattern(t *testing.T) {
	t.Parallel()

	t.Run("縦向きドアパターン", func(t *testing.T) {
		t.Parallel()
		// ###
		//  +   ← (1,1) が床、左右が壁、上下が床
		// ###
		mp := newTestMetaPlanForDoor()
		// 上下を床にする
		mp.Tiles[0*10+1] = oapi.Tile{Name: "floor", BlockPass: false}
		mp.Tiles[1*10+1] = oapi.Tile{Name: "floor", BlockPass: false}
		mp.Tiles[2*10+1] = oapi.Tile{Name: "floor", BlockPass: false}

		assert.True(t, isDoorPattern(mp, 1, 1, 10))
	})

	t.Run("横向きドアパターン", func(t *testing.T) {
		t.Parallel()
		// # #
		//  +   ← (1,1) が床、上下が壁、左右が床
		// # #
		mp := newTestMetaPlanForDoor()
		mp.Tiles[1*10+0] = oapi.Tile{Name: "floor", BlockPass: false}
		mp.Tiles[1*10+1] = oapi.Tile{Name: "floor", BlockPass: false}
		mp.Tiles[1*10+2] = oapi.Tile{Name: "floor", BlockPass: false}

		assert.True(t, isDoorPattern(mp, 1, 1, 10))
	})

	t.Run("パターンに一致しない", func(t *testing.T) {
		t.Parallel()
		// 全周が壁の床タイルはドアパターンではない
		mp := newTestMetaPlanForDoor()
		mp.Tiles[1*10+1] = oapi.Tile{Name: "floor", BlockPass: false}

		assert.False(t, isDoorPattern(mp, 1, 1, 10))
	})
}
