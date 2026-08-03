package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRectRoomPlanner_RoomsWithinBounds は生成した部屋の Max が inclusive 規約の
// width-1 / height-1 を超えないことを検証する。右端・下端付近に生成された部屋の Max が
// width/height に達すると、RoomDraw の inclusive ループが x==width へ到達し、CoordToIndex が
// 次行の column 0 へ折り返して孤立床タイルを生む。小さいマップと多数の seed で端の部屋を確実に踏む。
func TestRectRoomPlanner_RoomsWithinBounds(t *testing.T) {
	t.Parallel()

	width, height := consts.Tile(10), consts.Tile(10)
	for seed := range uint64(100) {
		chain, err := NewSmallRoomPlanner(width, height, seed)
		require.NoError(t, err)

		RectRoomPlanner{}.PlanRooms(&chain.PlanData)

		for i, room := range chain.PlanData.Rooms {
			assert.Lessf(t, int(room.Max.X), int(width),
				"seed=%d room=%d: Max.X=%d が width=%d 以上。inclusive 規約では width-1 まで",
				seed, i, int(room.Max.X), int(width))
			assert.Lessf(t, int(room.Max.Y), int(height),
				"seed=%d room=%d: Max.Y=%d が height=%d 以上",
				seed, i, int(room.Max.Y), int(height))
		}
	}
}
