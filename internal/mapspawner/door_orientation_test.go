package mapspawner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

// newTestMetaPlan はテスト用のMetaPlanデータを生成する。純粋なデータ生成のみ行う
func newTestMetaPlan(width, height int, tiles []oapi.Tile) *mapplanner.MetaPlan {
	return &mapplanner.MetaPlan{
		Level: gc.Level{
			TileWidth:  consts.Tile(width),
			TileHeight: consts.Tile(height),
		},
		Tiles: tiles,
	}
}

func TestDetectDoorOrientation_VerticalDoor(t *testing.T) {
	t.Parallel()

	// 3x3マップ: 左右が壁 → 縦向き
	// WWW
	// WDW
	// WWW
	tiles := make([]oapi.Tile, 9)
	for i := range tiles {
		tiles[i] = oapi.Tile{BlockPass: true}
	}
	tiles[4] = oapi.Tile{BlockPass: false} // 中央がドア

	plan := newTestMetaPlan(3, 3, tiles)
	result := detectDoorOrientation(plan, 1, 1)
	assert.Equal(t, gc.DoorOrientationVertical, result, "左右が壁なら縦向き")
}

func TestDetectDoorOrientation_HorizontalDoor(t *testing.T) {
	t.Parallel()

	// 3x3マップ: 上下が壁 → 横向き
	// .W.
	// .D.
	// .W.
	tiles := make([]oapi.Tile, 9)
	tiles[1] = oapi.Tile{BlockPass: true} // 上
	tiles[7] = oapi.Tile{BlockPass: true} // 下

	plan := newTestMetaPlan(3, 3, tiles)
	result := detectDoorOrientation(plan, 1, 1)
	assert.Equal(t, gc.DoorOrientationHorizontal, result, "上下が壁なら横向き")
}

func TestDetectDoorOrientation_EdgePosition(t *testing.T) {
	t.Parallel()

	tiles := make([]oapi.Tile, 9)
	plan := newTestMetaPlan(3, 3, tiles)

	// 端の座標はデフォルト（横向き）
	assert.Equal(t, gc.DoorOrientationHorizontal, detectDoorOrientation(plan, 0, 1))
	assert.Equal(t, gc.DoorOrientationHorizontal, detectDoorOrientation(plan, 2, 1))
	assert.Equal(t, gc.DoorOrientationHorizontal, detectDoorOrientation(plan, 1, 0))
	assert.Equal(t, gc.DoorOrientationHorizontal, detectDoorOrientation(plan, 1, 2))
}
