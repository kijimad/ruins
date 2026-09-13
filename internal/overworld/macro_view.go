package overworld

import "github.com/kijimaD/ruins/internal/consts"

// マクロ地図の描画モデル。1チャンク=1セルの地形俯瞰を、描画基盤に依らない形で表す。
// 全画面の俯瞰図も HUD の右上地図も、この同じモデルを各自の基盤で描く。ピクセルの描き方は
// 共有せず、どのチャンクに何が立ちマーカーが表示範囲のどこに来るかというモデルだけを共有する。

// MacroMargin は帯の南北へ足す余白チャンク数。北はこの先の地形を先読みし、南は通った地形を見返せる。
// 全画面図も HUD もこの余白ぶん広い表示範囲を描く。
const MacroMargin consts.Chunk = 6

// MacroRange はマクロ地図の表示範囲。左上端の絶対チャンク座標と、表示範囲の列数・行数をチャンク単位で持つ。
type MacroRange struct {
	OriginX consts.Chunk // 表示範囲の左端の絶対チャンク列
	OriginY consts.Chunk // 表示範囲の上端の絶対チャンク行
	Cols    consts.Chunk // 表示範囲の列数。帯の Cols と同じ有界幅
	Rows    consts.Chunk // 表示範囲の行数
}

// FullBandRange は帯全体を覆う表示範囲を帯のプリミティブから組む。南北へ MacroMargin ぶん広げ、
// この先の地形を先読みできる。全画面の地形俯瞰図が使う。北は -Y なので北端は -northIndex。
func FullBandRange(northIndex, cols, rows consts.Chunk) MacroRange {
	return MacroRange{
		OriginX: 0,
		OriginY: -northIndex - MacroMargin,
		Cols:    max(cols, 1),
		Rows:    rows + 2*MacroMargin,
	}
}

// PlayerCenteredRange はプレイヤーの絶対チャンク行を中心に、南北へ radius チャンクぶんの表示範囲を組む。
// 横は帯全体を見せる。HUD の右上地図が近傍だけを大きく描くために使う。
func PlayerCenteredRange(centerRow, cols consts.Chunk, radius int) MacroRange {
	return MacroRange{
		OriginX: 0,
		OriginY: centerRow - consts.Chunk(radius),
		Cols:    max(cols, 1),
		Rows:    consts.Chunk(2*radius + 1),
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
// ChunkPlace を表示範囲の全チャンクへ適用し、マーカーはタイル座標をチャンク寸法で割って表示範囲ローカルへ移す。
// northIndex は帯ローカルなタイル座標を絶対チャンク行へ移すのに使う。X は有界なので列は割るだけ。
// discovered は開放済みチャンクの集合で、絶対チャンク座標をキーにする。含まれるチャンクだけを開放し、
// 残りはフォグで伏せる。nil や空集合は「まだ何も開放していない」を表す。Go の nil マップ読み取りは
// 安全に false を返すので、nil でも全チャンクがフォグになる。
func BuildMacroView(
	runSeed uint64,
	northIndex consts.Chunk,
	chunkW, chunkH consts.Tile,
	area MacroRange,
	playerTile consts.Coord[consts.Tile],
	hasPlayer bool,
	cubeTiles []consts.Coord[consts.Tile],
	discovered map[consts.Coord[consts.Chunk]]bool,
) MacroView {
	cols := max(area.Cols, 1)
	rows := max(area.Rows, 1)

	// 道の接続方角を表示範囲で先に算出する。種別記号と同じく生成を伴わない純関数。
	// ChunkPlace/道の有界カウントは帯の列数で、表示範囲の Cols がそれに相当する
	roads := buildRoadOverlay(runSeed, area, cols)

	cells := make([][]MacroCell, rows)
	for cy := range rows {
		cells[cy] = make([]MacroCell, cols)
		for i := range cols {
			c := consts.Coord[consts.Chunk]{X: area.OriginX + i, Y: area.OriginY + cy}
			cells[cy][i] = MacroCell{
				Glyph:      ChunkPlace(runSeed, c, cols),
				Discovered: discovered[c],
				Road:       roads[c],
			}
		}
	}

	// toCell は帯ローカルなタイル座標を表示範囲ローカルのチャンクセルへ移す。表示範囲外なら ok=false。
	// chunkW/chunkH は帯の1チャンクのタイル寸法で、帯が有効なら必ず正なのでゼロ除算しない。
	// X は有界なので絶対チャンク列はタイルを幅で割るだけ。Y は北進ぶん負へずらした絶対チャンク行。
	// 帯ローカルの t は非負なので素の整数除算で足り、負座標用の FloorDiv は要らない
	toCell := func(t consts.Coord[consts.Tile]) (consts.Coord[consts.Chunk], bool) {
		worldCol := consts.Chunk(int(t.X) / int(chunkW))
		worldRow := consts.Chunk(int(t.Y)/int(chunkH)) - northIndex
		col := worldCol - area.OriginX
		row := worldRow - area.OriginY
		if col >= 0 && col < cols && row >= 0 && row < rows {
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
