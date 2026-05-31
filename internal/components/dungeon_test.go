package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestNewDungeon(t *testing.T) {
	t.Parallel()

	d := NewDungeon()

	assert.NotNil(t, d.ExploredTiles, "ExploredTilesが初期化されている")
	assert.Equal(t, 1, d.SelectedWeaponSlot, "初期武器スロットは1")
	assert.Equal(t, 150, d.MinimapSettings.Width)
	assert.Equal(t, 150, d.MinimapSettings.Height)
	assert.Equal(t, 3, d.MinimapSettings.Scale)
}

func TestLevel_XYTileIndex(t *testing.T) {
	t.Parallel()

	level := &Level{TileWidth: 10, TileHeight: 5}

	tests := []struct {
		name     string
		tx, ty   consts.Tile
		expected TileIdx
	}{
		{"左上", 0, 0, 0},
		{"1行目の2番目", 1, 0, 1},
		{"2行目の先頭", 0, 1, 10},
		{"右下", 9, 4, 49},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, level.XYTileIndex(tt.tx, tt.ty))
		})
	}
}

func TestLevel_XYTileCoord(t *testing.T) {
	t.Parallel()

	level := &Level{TileWidth: 10, TileHeight: 5}

	tests := []struct {
		name      string
		idx       TileIdx
		expectedX consts.Pixel
		expectedY consts.Pixel
	}{
		{"インデックス0は左上", 0, 0, 0},
		{"インデックス1は1列目", 1, 1, 0},
		{"インデックス10は2行目先頭", 10, 0, 1},
		{"インデックス49は右下", 49, 9, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			x, y := level.XYTileCoord(tt.idx)
			assert.Equal(t, tt.expectedX, x)
			assert.Equal(t, tt.expectedY, y)
		})
	}
}

func TestLevel_XYTileIndex_and_XYTileCoord_roundtrip(t *testing.T) {
	t.Parallel()

	level := &Level{TileWidth: 10, TileHeight: 5}

	for ty := consts.Tile(0); ty < level.TileHeight; ty++ {
		for tx := consts.Tile(0); tx < level.TileWidth; tx++ {
			idx := level.XYTileIndex(tx, ty)
			gotX, gotY := level.XYTileCoord(idx)
			assert.Equal(t, consts.Pixel(tx), gotX)
			assert.Equal(t, consts.Pixel(ty), gotY)
		}
	}
}
