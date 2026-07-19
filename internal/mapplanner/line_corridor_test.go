package mapplanner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestLineCorridorPlanner_NarrowAtRoomBoundary(t *testing.T) {
	t.Parallel()

	t.Run("廊下が部屋境界で1タイル幅に狭まる", func(t *testing.T) {
		t.Parallel()

		const width, height = 20, 20
		tiles := make([]oapi.Tile, width*height)
		for i := range tiles {
			tiles[i] = oapi.Tile{Name: "wall", BlockPass: true}
		}

		// 2つの部屋を縦に配置する。廊下が上下に接続される
		room1 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 8, Y: 2}, Max: consts.Coord[consts.Tile]{X: 12, Y: 5}}
		room2 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 8, Y: 14}, Max: consts.Coord[consts.Tile]{X: 12, Y: 17}}
		rooms := []gc.Rect{room1, room2}

		// 部屋の内部を床にする
		for _, room := range rooms {
			for x := int(room.Min.X); x <= int(room.Max.X); x++ {
				for y := int(room.Min.Y); y <= int(room.Max.Y); y++ {
					tiles[y*width+x] = oapi.Tile{Name: "floor", BlockPass: false}
				}
			}
		}

		mp := &MetaPlan{
			Level: gc.Level{
				TileWidth:  consts.Tile(width),
				TileHeight: consts.Tile(height),
			},
			Tiles:     tiles,
			Rooms:     rooms,
			RNG:       rand.New(rand.NewPCG(42, 43)),
			RawMaster: CreateTestRawMaster(),
		}

		planner := LineCorridorPlanner{}
		planner.BuildCorridors(mp)

		centerX, _ := room1.Center()

		// 部屋の境界付近(y=6: room1の下辺y=5の直下)で1タイル幅を確認
		boundaryY := int(room1.Max.Y) + 1
		floorCount := 0
		for x := int(centerX) - 1; x <= int(centerX)+1; x++ {
			if x >= 0 && x < width {
				idx := boundaryY*width + x
				if !mp.Tiles[idx].BlockPass {
					floorCount++
				}
			}
		}
		assert.Equal(t, 1, floorCount,
			"部屋境界付近で廊下は1タイル幅であるべき（実際: %d）", floorCount)

		// 部屋から離れた中間地点では3タイル幅であることを確認
		_, centerY1 := room1.Center()
		_, centerY2 := room2.Center()
		midY := (int(centerY1) + int(centerY2)) / 2
		midFloorCount := 0
		for x := int(centerX) - 1; x <= int(centerX)+1; x++ {
			if x >= 0 && x < width {
				idx := midY*width + x
				if !mp.Tiles[idx].BlockPass {
					midFloorCount++
				}
			}
		}
		assert.Equal(t, 3, midFloorCount,
			"部屋から離れた中間地点では廊下は3タイル幅であるべき（実際: %d）", midFloorCount)
	})
}

func TestIsAdjacentToRoom(t *testing.T) {
	t.Parallel()

	rooms := []gc.Rect{{Min: consts.Coord[consts.Tile]{X: 3, Y: 3}, Max: consts.Coord[consts.Tile]{X: 6, Y: 6}}}

	tests := []struct {
		name     string
		x, y     int
		expected bool
	}{
		{"部屋内", 4, 4, true},
		{"部屋の上辺に隣接", 4, 2, true},
		{"部屋の左辺に隣接", 2, 4, true},
		{"部屋の角に斜めで隣接", 2, 2, true},
		{"部屋から2タイル離れている", 1, 4, false},
		{"部屋から2タイル斜めに離れている", 1, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, isAdjacentToRoom(rooms, tt.x, tt.y))
		})
	}
}
