package hud

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMinimap(t *testing.T) *Minimap {
	t.Helper()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	return NewMinimap(nil, NewChrome(res))
}

func TestMinimap_Draw_無効なら何も描かない(t *testing.T) {
	t.Parallel()
	minimap := newTestMinimap(t)
	minimap.enabled = false
	cv := &fakeCanvas{}

	minimap.Draw(cv, MinimapData{
		MinimapConfig:    MinimapConfig{Width: 150, Height: 150, Scale: 3},
		ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, 0, cv.nineSlices)
	assert.Empty(t, cv.texts)
	assert.Empty(t, cv.fillRects)
}

func TestMinimap_Draw_探索済みタイルがなければNoDataを表示する(t *testing.T) {
	t.Parallel()
	minimap := newTestMinimap(t)
	cv := &fakeCanvas{}

	minimap.Draw(cv, MinimapData{
		MinimapConfig:    MinimapConfig{Width: 150, Height: 150, Scale: 3},
		ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, 1, cv.nineSlices, "背景パネルを1回描く")
	require.NotEmpty(t, cv.texts)
	assert.Equal(t, "No Data", cv.texts[len(cv.texts)-1].str)
	assert.Empty(t, cv.fillRects, "タイルが無いので塗りつぶしは無い")
}

func TestMinimap_Draw_サイズ0なら探索済みタイルがあっても背景パネルを描かない(t *testing.T) {
	t.Parallel()
	minimap := newTestMinimap(t)
	cv := &fakeCanvas{}

	tile := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
	minimap.Draw(cv, MinimapData{
		ExploredTiles:    map[gc.GridElement]bool{tile: true},
		MinimapConfig:    MinimapConfig{Width: 0, Height: 0, Scale: 3},
		ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, 0, cv.nineSlices, "サイズ0のときは背景を敷かない")
}

func TestMinimap_Draw_探索済みタイルをプレイヤー位置基準で描く(t *testing.T) {
	t.Parallel()
	minimap := newTestMinimap(t)
	cv := &fakeCanvas{}

	playerTile := consts.Coord[consts.Tile]{X: 10, Y: 10}
	inRange := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}}
	outOfRange := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1000, Y: 1000}}

	data := MinimapData{
		PlayerTile: playerTile,
		ExploredTiles: map[gc.GridElement]bool{
			inRange:    true,
			outOfRange: true,
		},
		TileColors: map[gc.GridElement]TileColorInfo{
			inRange:    {R: 10, G: 20, B: 30, A: 255},
			outOfRange: {R: 1, G: 2, B: 3, A: 255},
		},
		MinimapConfig:    MinimapConfig{Width: 150, Height: 150, Scale: 3},
		ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
	}
	minimap.Draw(cv, data)

	assert.Equal(t, 1, cv.nineSlices, "背景パネルを描く")
	assert.Len(t, cv.fillRects, 2, "範囲内のタイル1つとプレイヤーの印だけ描く。範囲外のタイルは除外する")
}
