package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoomDraw_マップ角に接する部屋の角タイルもfloor化する は、部屋がマップの
// 左上角や右下角に密着したとき、フラットインデックスの先頭 idx==0 と末尾
// idx==W*H-1 に当たる角タイルも floor 化されることを確認する。
// かつて境界チェックが 0 < idx && idx < W*H-1 で、この2点だけを誤って
// 除外し孤立した壁を残していた回帰を防ぐ。
func TestRoomDraw_マップ角に接する部屋の角タイルもfloor化する(t *testing.T) {
	t.Parallel()

	const width, height = consts.Tile(5), consts.Tile(5)

	tests := []struct {
		name string
		room gc.Rect
		// cornerIdx は floor 化を確認したい角のフラットインデックス
		cornerIdx gc.TileIdx
	}{
		{
			name:      "左上角に密着する部屋の角idx0がfloorになる",
			room:      gc.Rect{Min: consts.Coord[consts.Tile]{X: 0, Y: 0}, Max: consts.Coord[consts.Tile]{X: 2, Y: 2}},
			cornerIdx: 0,
		},
		{
			name:      "右下角に密着する部屋の角idxW*H-1がfloorになる",
			room:      gc.Rect{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}},
			cornerIdx: gc.TileIdx(width*height - 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			planData := &MetaPlan{
				Level:     gc.Level{TileWidth: width, TileHeight: height},
				Tiles:     make([]oapi.Tile, int(width)*int(height)),
				Rooms:     []gc.Rect{tt.room},
				RawMaster: CreateTestRawMaster(),
			}
			for i := range planData.Tiles {
				planData.Tiles[i] = planData.GetTile("wall")
			}

			require.NoError(t, RoomDraw{}.PlanMeta(planData))

			assert.Equal(t, consts.TileNameFloor, planData.Tiles[tt.cornerIdx].Id)
		})
	}
}
