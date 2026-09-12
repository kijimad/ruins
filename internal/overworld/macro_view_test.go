package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestFullBandRange_帯全体に南北の余白を足す(t *testing.T) {
	t.Parallel()
	area := FullBandRange(10, 4, 3)
	assert.Equal(t, consts.Chunk(0), area.OriginX, "表示範囲の左端は帯左端の0")
	assert.Equal(t, consts.Chunk(4), area.Cols, "列数は帯の有界幅そのまま")
	assert.Equal(t, -10-MacroMargin, area.OriginY, "表示範囲の上端は北進位置から余白ぶん北。北は -Y")
	assert.Equal(t, 3+2*MacroMargin, area.Rows, "行数は帯高に南北の余白を足す")
}

func TestBuildMacroView_表示範囲の格子とマーカーを表示範囲ローカルへ組む(t *testing.T) {
	t.Parallel()
	// eastIndex=0, 1チャンク=10タイル。表示範囲は絶対列0から3列×2行
	area := MacroRange{OriginX: 0, Cols: 3, Rows: 2}
	player := consts.Coord[consts.Tile]{X: 15, Y: 5}     // チャンク(1,0)
	cubes := []consts.Coord[consts.Tile]{{X: 25, Y: 12}} // チャンク(2,1)
	view := BuildMacroView(1, 0, 10, 10, area, player, true, cubes, nil)

	assert.Len(t, view.Cells, 2, "行数は表示範囲の Rows")
	assert.Len(t, view.Cells[0], 3, "列数は表示範囲の Cols")
	assert.Equal(t, consts.Coord[consts.Chunk]{X: 1, Y: 0}, view.PlayerCell, "プレイヤーは表示範囲ローカル(1,0)")
	assert.Equal(t, []consts.Coord[consts.Chunk]{{X: 2, Y: 1}}, view.CubeCells, "キューブは表示範囲ローカル(2,1)")
	assert.False(t, view.Cells[0][0].Discovered, "discovered が nil なら何も開放されずフォグになる")
}

func TestBuildMacroView_フォグは探索済みチャンクだけ開放する(t *testing.T) {
	t.Parallel()
	area := MacroRange{OriginX: 0, Cols: 3, Rows: 1}
	// 絶対チャンク(1,0)だけ開放済みにする
	discovered := map[consts.Coord[consts.Chunk]]bool{{X: 1, Y: 0}: true}
	view := BuildMacroView(1, 0, 10, 10, area, consts.Coord[consts.Tile]{}, false, nil, discovered)

	assert.False(t, view.Cells[0][0].Discovered, "未探索チャンクは伏せる")
	assert.True(t, view.Cells[0][1].Discovered, "探索済みチャンクは開放する")
	assert.False(t, view.Cells[0][2].Discovered, "未探索チャンクは伏せる")
}

func TestBuildMacroView_道の接続方角をセルに埋める(t *testing.T) {
	t.Parallel()
	const runSeed uint64 = 12345
	const rows consts.Chunk = 9
	// 複数リージョンを覆う表示範囲なら街道が必ず通る。道セルは Road のビットが立つ
	area := MacroRange{OriginX: 0, Cols: 3 * settlementPlacement.Spacing, Rows: rows}
	view := BuildMacroView(runSeed, 0, 10, 10, area, consts.Coord[consts.Tile]{}, false, nil, nil)

	found := false
	for _, row := range view.Cells {
		for _, cell := range row {
			if cell.Road.Any() {
				found = true
				assert.Zero(t, cell.Road&^(RoadN|RoadS|RoadE|RoadW), "未定義ビットは立たない")
			}
		}
	}
	assert.True(t, found, "複数リージョンを覆う表示範囲には道セルがある")
}

func TestBuildMacroView_表示範囲外のマーカーは落とす(t *testing.T) {
	t.Parallel()
	area := MacroRange{OriginX: 0, Cols: 2, Rows: 1}
	player := consts.Coord[consts.Tile]{X: 55, Y: 5} // チャンク(5,0)。表示範囲の外
	view := BuildMacroView(1, 0, 10, 10, area, player, true, []consts.Coord[consts.Tile]{{X: 99, Y: 99}}, nil)

	assert.Equal(t, consts.Chunk(-1), view.PlayerCell.X, "表示範囲外のプレイヤーは -1")
	assert.Empty(t, view.CubeCells, "表示範囲外のキューブは載せない")
}

func TestBuildMacroView_プレイヤー不在なら現在地なし(t *testing.T) {
	t.Parallel()
	area := MacroRange{OriginX: 0, Cols: 2, Rows: 1}
	view := BuildMacroView(1, 0, 10, 10, area, consts.Coord[consts.Tile]{}, false, nil, nil)

	assert.Equal(t, consts.Chunk(-1), view.PlayerCell.X, "プレイヤー不在なら -1")
}
