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

func TestSeamlessBand_前線ジオメトリ(t *testing.T) {
	t.Parallel()

	// EastIndex=1, ChunkW=40 → 帯原点は絶対40。前線東端60・幅20 → ゾーンは (40, 60]
	sb := SeamlessBand{EastIndex: 1, ChunkW: 40, FrontEastAbsX: 60, FrontColdWidth: 20}

	assert.Equal(t, consts.Tile(40), sb.BandOriginX(), "帯原点 = EastIndex*ChunkW")
	assert.Equal(t, consts.Tile(50), sb.LocalToAbsX(10), "ローカル10 = 絶対50")
	assert.Equal(t, consts.Tile(40), sb.ColdZoneWestAbsX(), "西端 = FrontEast - ColdWidth")

	assert.False(t, sb.InColdZone(40), "西端は含まない（進入不可ライン）")
	assert.True(t, sb.InColdZone(41), "ゾーン内")
	assert.True(t, sb.InColdZone(60), "東端は含む")
	assert.False(t, sb.InColdZone(61), "前線より東は平常")

	assert.True(t, sb.IsWestOfFrontLine(40), "西端ちょうどは進入不可側")
	assert.True(t, sb.IsWestOfFrontLine(30), "西は進入不可側")
	assert.False(t, sb.IsWestOfFrontLine(50), "ゾーン内は進入不可側でない")
}
