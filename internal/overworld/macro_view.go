package overworld

import "github.com/kijimaD/ruins/internal/consts"

// マクロ地図の描画モデル。1チャンク=1セルの地形俯瞰を、描画基盤に依らない形で表す。
// 全画面の俯瞰図も HUD の右上地図も、この同じモデルを各自の基盤で描く。ピクセルの描き方は
// 共有せず、どのチャンクに何が立ちマーカーが表示範囲のどこに来るかというモデルだけを共有する。

// MacroMargin は帯の東西へ足す余白チャンク数。この先の地形を先読みできる。全画面図も HUD も
// この余白ぶん広い表示範囲を描く。
const MacroMargin consts.Chunk = 6

// MacroRange はマクロ地図の表示範囲。左端の絶対チャンク列と、表示範囲の列数・行数をチャンク単位で持つ。
type MacroRange struct {
	OriginX consts.Chunk // 表示範囲の左端の絶対チャンク列
	Cols    consts.Chunk // 表示範囲の列数
	Rows    consts.Chunk // 表示範囲の行数。帯の Rows と同じ
}

// FullBandRange は帯全体を覆う表示範囲を帯のプリミティブから組む。東西へ MacroMargin ぶん広げ、
// この先の地形を先読みできる。全画面の地形俯瞰図が使う。
func FullBandRange(eastIndex, cols, rows consts.Chunk) MacroRange {
	return MacroRange{
		OriginX: eastIndex - MacroMargin,
		Cols:    cols + 2*MacroMargin,
		Rows:    max(rows, 1),
	}
}

// PlayerCenteredRange はプレイヤーの絶対チャンク列を中心に、左右へ radius チャンクぶんの表示範囲を組む。
// 縦は帯全体を見せる。HUD の右上地図が近傍だけを大きく描くために使う。
func PlayerCenteredRange(centerCol, rows consts.Chunk, radius int) MacroRange {
	return MacroRange{
		OriginX: centerCol - consts.Chunk(radius),
		Cols:    consts.Chunk(2*radius + 1),
		Rows:    max(rows, 1),
	}
}

// MacroCell は表示範囲内1チャンクの表示情報。種別文字と、探索で開放済みかを持つ。色は文字から引く。
type MacroCell struct {
	Glyph      rune
	Discovered bool    // このチャンクが探索で開放済みか。未開放は伏せてフォグにする
	Road       RoadDir // このチャンクを通る道の接続方角。0 なら道なし。線分描画でセル中央から辺へ引く
}

// MacroView はマクロ地図の描画モデル。表示範囲内のチャンク格子と、マーカーの表示範囲ローカル座標を持つ。
type MacroView struct {
	Cells      [][]MacroCell                // [row][col] の種別文字格子
	PlayerCell consts.Coord[consts.Chunk]   // 表示範囲ローカル (列,行)。表示範囲外や不在なら X が -1
	CubeCells  []consts.Coord[consts.Chunk] // 表示範囲ローカルのキューブ位置。表示範囲内のものだけ
}

// BuildMacroView は帯のプリミティブとプレイヤー・キューブのタイル座標から、指定表示範囲の描画モデルを組む。
// ChunkPlace を表示範囲の全チャンクへ適用し、マーカーはタイル座標をチャンク幅で割って表示範囲ローカルへ移す。
// eastIndex は帯ローカルなタイル座標を絶対チャンク列へ移すのに使う。
// discovered は開放済みチャンクの集合で、絶対チャンク座標をキーにする。含まれるチャンクだけを開放し、
// 残りはフォグで伏せる。nil や空集合は「まだ何も開放していない」を表す。Go の nil マップ読み取りは
// 安全に false を返すので、nil でも全チャンクがフォグになる。
func BuildMacroView(
	runSeed uint64,
	eastIndex consts.Chunk,
	chunkW, chunkH consts.Tile,
	area MacroRange,
	playerTile consts.Coord[consts.Tile],
	hasPlayer bool,
	cubeTiles []consts.Coord[consts.Tile],
	discovered map[consts.Coord[consts.Chunk]]bool,
) MacroView {
	rows := max(area.Rows, 1)

	// 道の接続方角を表示範囲で先に算出する。種別記号と同じく生成を伴わない純関数
	roads := buildRoadOverlay(runSeed, area, rows)

	cells := make([][]MacroCell, rows)
	for cy := range rows {
		cells[cy] = make([]MacroCell, area.Cols)
		for i := range area.Cols {
			c := consts.Coord[consts.Chunk]{X: area.OriginX + i, Y: cy}
			cells[cy][i] = MacroCell{
				Glyph:      ChunkPlace(runSeed, c, rows),
				Discovered: discovered[c],
				Road:       roads[c],
			}
		}
	}

	// toCell は帯ローカルなタイル座標を表示範囲ローカルのチャンクセルへ移す。表示範囲外なら ok=false。
	// chunkW/chunkH は帯の1チャンクのタイル寸法で、帯が有効なら必ず正なのでゼロ除算しない。
	toCell := func(t consts.Coord[consts.Tile]) (consts.Coord[consts.Chunk], bool) {
		worldCol := eastIndex + consts.Chunk(int(t.X)/int(chunkW))
		col := worldCol - area.OriginX
		row := consts.Chunk(int(t.Y) / int(chunkH))
		if col >= 0 && col < area.Cols && row >= 0 && row < rows {
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
