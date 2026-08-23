package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestNewStageField(t *testing.T) {
	t.Parallel()

	m := NewStageField()

	assert.NotNil(t, m.ExploredTiles, "ExploredTilesが初期化されている")
}

func TestLevel_CoordToIndex(t *testing.T) {
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
			assert.Equal(t, tt.expected, level.CoordToIndex(consts.Coord[consts.Tile]{X: tt.tx, Y: tt.ty}))
		})
	}
}

func TestLevel_IndexToCoord(t *testing.T) {
	t.Parallel()

	level := &Level{TileWidth: 10, TileHeight: 5}

	tests := []struct {
		name      string
		idx       TileIdx
		expectedX consts.Tile
		expectedY consts.Tile
	}{
		{"インデックス0は左上", 0, 0, 0},
		{"インデックス1は1列目", 1, 1, 0},
		{"インデックス10は2行目先頭", 10, 0, 1},
		{"インデックス49は右下", 49, 9, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pos := level.IndexToCoord(tt.idx)
			assert.Equal(t, tt.expectedX, pos.X)
			assert.Equal(t, tt.expectedY, pos.Y)
		})
	}
}

func TestLevel_CoordToIndex_and_IndexToCoord_roundtrip(t *testing.T) {
	t.Parallel()

	level := &Level{TileWidth: 10, TileHeight: 5}

	for ty := consts.Tile(0); ty < level.TileHeight; ty++ {
		for tx := consts.Tile(0); tx < level.TileWidth; tx++ {
			idx := level.CoordToIndex(consts.Coord[consts.Tile]{X: tx, Y: ty})
			assert.Equal(t, consts.Coord[consts.Tile]{X: tx, Y: ty}, level.IndexToCoord(idx))
		}
	}
}

func TestSeamlessBand_座標変換(t *testing.T) {
	t.Parallel()

	// EastIndex=1, ChunkW=40 → 帯原点は絶対40
	sb := SeamlessBand{EastIndex: 1, ChunkW: 40}

	assert.Equal(t, consts.AbsTileX(40), sb.BandOriginX(), "帯原点 = EastIndex*ChunkW")
	assert.Equal(t, consts.AbsTileX(50), sb.LocalToAbsX(10), "ローカル10 = 絶対50")
}
