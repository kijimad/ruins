package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestFullBandWindow_帯全体に東西の余白を足す(t *testing.T) {
	t.Parallel()
	win := FullBandWindow(10, 4, 3)
	assert.Equal(t, 10-MacroMargin, win.OriginX, "窓左端は東進位置から余白ぶん西")
	assert.Equal(t, int(4+2*MacroMargin), win.Cols, "列数は帯幅に東西の余白を足す")
	assert.Equal(t, consts.Chunk(3), win.Rows, "行数は帯の高さ")
}

func TestBuildMacroView_窓の格子とマーカーを窓ローカルへ組む(t *testing.T) {
	t.Parallel()
	// eastIndex=0, 1チャンク=10タイル。窓は絶対列0から3列×2行
	win := MacroWindow{OriginX: 0, Cols: 3, Rows: 2}
	player := consts.Coord[consts.Tile]{X: 15, Y: 5}     // チャンク(1,0)
	cubes := []consts.Coord[consts.Tile]{{X: 25, Y: 12}} // チャンク(2,1)
	view := BuildMacroView(1, 0, 10, 10, win, player, true, cubes)

	assert.Len(t, view.Cells, 2, "行数は窓の Rows")
	assert.Len(t, view.Cells[0], 3, "列数は窓の Cols")
	assert.Equal(t, consts.Coord[consts.Chunk]{X: 1, Y: 0}, view.PlayerCell, "プレイヤーは窓ローカル(1,0)")
	assert.Equal(t, []consts.Coord[consts.Chunk]{{X: 2, Y: 1}}, view.CubeCells, "キューブは窓ローカル(2,1)")
}

func TestBuildMacroView_窓外のマーカーは落とす(t *testing.T) {
	t.Parallel()
	win := MacroWindow{OriginX: 0, Cols: 2, Rows: 1}
	player := consts.Coord[consts.Tile]{X: 55, Y: 5} // チャンク(5,0)。窓の外
	view := BuildMacroView(1, 0, 10, 10, win, player, true, []consts.Coord[consts.Tile]{{X: 99, Y: 99}})

	assert.Equal(t, consts.Chunk(-1), view.PlayerCell.X, "窓外のプレイヤーは -1")
	assert.Empty(t, view.CubeCells, "窓外のキューブは載せない")
}

func TestBuildMacroView_プレイヤー不在なら現在地なし(t *testing.T) {
	t.Parallel()
	win := MacroWindow{OriginX: 0, Cols: 2, Rows: 1}
	view := BuildMacroView(1, 0, 10, 10, win, consts.Coord[consts.Tile]{}, false, nil)

	assert.Equal(t, consts.Chunk(-1), view.PlayerCell.X, "プレイヤー不在なら -1")
}
