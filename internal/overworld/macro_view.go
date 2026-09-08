package overworld

import "github.com/kijimaD/ruins/internal/consts"

// マクロ地図の描画モデル。1チャンク=1セルの地形俯瞰を、描画基盤に依らない形で表す。
// 全画面の俯瞰図も HUD の右上地図も、この同じモデルを各自の基盤で描く。ピクセルの描き方は
// 共有せず、どのチャンクに何が立ちマーカーが窓のどこに来るかというモデルだけを共有する。

// MacroMargin は帯の東西へ足す余白チャンク数。この先の地形を先読みできる。全画面図も HUD も
// この余白ぶん広い窓を描く。
const MacroMargin consts.Chunk = 6

// MacroWindow はマクロ地図の窓。左端の絶対チャンク列と、窓の列数・行数をチャンク単位で持つ。
type MacroWindow struct {
	OriginX consts.Chunk // 窓左端の絶対チャンク列
	Cols    int          // 窓の列数
	Rows    consts.Chunk // 窓の行数。帯の Rows と同じ
}

// FullBandWindow は帯全体を覆う窓を帯のプリミティブから組む。東西へ MacroMargin ぶん広げ、
// この先の地形を先読みできる。全画面図も HUD もこの同じ窓を使う。
func FullBandWindow(eastIndex, cols, rows consts.Chunk) MacroWindow {
	return MacroWindow{
		OriginX: eastIndex - MacroMargin,
		Cols:    int(cols + 2*MacroMargin),
		Rows:    max(rows, 1),
	}
}

// MacroCell は窓内1チャンクの表示情報。今は種別文字だけを持つ。色は文字から引く。
type MacroCell struct {
	Glyph rune
}

// MacroView はマクロ地図の描画モデル。窓内のチャンク格子と、マーカーの窓ローカル座標を持つ。
type MacroView struct {
	Cells      [][]MacroCell                // [row][col] の種別文字格子
	PlayerCell consts.Coord[consts.Chunk]   // 窓ローカル (列,行)。窓外や不在なら X が -1
	CubeCells  []consts.Coord[consts.Chunk] // 窓ローカルのキューブ位置。窓内のものだけ
}

// BuildMacroView は帯のプリミティブとプレイヤー・キューブのタイル座標から、指定窓の描画モデルを組む。
// ChunkPlace を窓の全チャンクへ適用し、マーカーはタイル座標をチャンク幅で割って窓ローカルへ移す。
// eastIndex は帯ローカルなタイル座標を絶対チャンク列へ移すのに使う。
func BuildMacroView(
	runSeed uint64,
	eastIndex consts.Chunk,
	chunkW, chunkH consts.Tile,
	win MacroWindow,
	playerTile consts.Coord[consts.Tile],
	hasPlayer bool,
	cubeTiles []consts.Coord[consts.Tile],
) MacroView {
	rows := max(win.Rows, 1)

	cells := make([][]MacroCell, rows)
	for cy := range rows {
		cells[cy] = make([]MacroCell, win.Cols)
		for i := range win.Cols {
			c := consts.Coord[consts.Chunk]{X: win.OriginX + consts.Chunk(i), Y: cy}
			cells[cy][i] = MacroCell{Glyph: ChunkPlace(runSeed, c, rows)}
		}
	}

	// toCell は帯ローカルなタイル座標を窓ローカルのチャンクセルへ移す。窓外なら ok=false。
	toCell := func(t consts.Coord[consts.Tile]) (consts.Coord[consts.Chunk], bool) {
		worldCol := eastIndex + consts.Chunk(int(t.X)/int(chunkW))
		col := worldCol - win.OriginX
		row := consts.Chunk(int(t.Y) / int(chunkH))
		if col >= 0 && int(col) < win.Cols && row >= 0 && row < rows {
			return consts.Coord[consts.Chunk]{X: col, Y: row}, true
		}
		return consts.Coord[consts.Chunk]{}, false
	}

	view := MacroView{Cells: cells, PlayerCell: consts.Coord[consts.Chunk]{X: -1}}
	if hasPlayer {
		if pc, ok := toCell(playerTile); ok {
			view.PlayerCell = pc
		}
	}
	for _, ct := range cubeTiles {
		if cc, ok := toCell(ct); ok {
			view.CubeCells = append(view.CubeCells, cc)
		}
	}
	return view
}
