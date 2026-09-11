package states

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// OverworldMapState はオーバーワールドの種別俯瞰図を全画面で表示するステート。
// 1チャンク=1建物の縮尺に合わせ、各チャンクを1マスの色と文字で示す。1マス=1つの場所。
// 生成は純関数なので ECS のタイルを生成せずに算出して描ける。
type OverworldMapState struct {
	es.BaseState[w.World]

	view      overworld.MacroView        // プレイヤー中心のチャンク俯瞰。glyph 格子とマーカー
	playerAbs consts.Coord[consts.Chunk] // 現在地の絶対チャンク座標。ヘッダ表示に使う
	cellPx    consts.ScreenPixel         // 1チャンクのセル寸法。表示範囲の半径の算出と描画で共有する
	body      uicore.Drawable            // モーダルのパネル。初回 Draw で1度組み以後描く
}

var _ es.State[w.World] = &OverworldMapState{}

// OnPause はステートが一時停止される際に呼ばれる。
func (st *OverworldMapState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる。
func (st *OverworldMapState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる。
func (st *OverworldMapState) OnStop(_ w.World) error { return nil }

// 全画面図のセル寸法・半径の範囲。帯が短いとセルが巨大化、広いと潰れるのを両側で防ぐ。
// セルを小さくするほどモーダル幅に多くのチャンクが収まり、見える範囲が広がる
const (
	overworldMapMinCell   = 10
	overworldMapMaxCell   = 18
	overworldMapMinRadius = 3
)

// modalInner はモーダルパネルの内側矩形を返す。表示範囲の半径・セル寸法の算出とパネル画像の寸法で共有する。
func (st *OverworldMapState) modalInner(world w.World) image.Rectangle {
	return menuframe.PanelInner(menuframe.ModalRect(world))
}

// overworldMapCell は帯の行数からセル寸法を決める。見出し・凡例のぶんを足した行数で内側高さを割り、
// 大きめのセルへ寄せる。帯が短いと巨大化するので上限で止める。
func overworldMapCell(inner image.Rectangle, rows consts.Chunk) consts.ScreenPixel {
	cell := min(max(inner.Dy()/(int(rows)+3), overworldMapMinCell), overworldMapMaxCell)
	return consts.ScreenPixel(cell)
}

// overworldMapRadius はモーダル幅に収まるプレイヤー左右のチャンク数を返す。最低限は確保する。
func overworldMapRadius(inner image.Rectangle, cell consts.ScreenPixel) int {
	cols := inner.Dx() / int(cell)
	return max((cols-1)/2, overworldMapMinRadius)
}

// OnStart はプレイヤー中心の地形俯瞰モデルを算出して保持する。表示中はプレイヤーが動かないため
// 一度だけ計算する。セル寸法と半径をモーダル寸法から決めてモーダルいっぱいに大きく見せ、モデル
// 生成は overworld.BuildMacroView に委ねる。フォグは HUD と同じで探索済みチャンクだけを開放する。
func (st *OverworldMapState) OnStart(world w.World) error {
	sb := query.GetSeamlessBand(world)
	if sb == nil || !sb.Active {
		return fmt.Errorf("overworld band is not valid")
	}
	playerTile, hasPlayer := query.PlayerBandTile(world)
	inner := st.modalInner(world)
	st.cellPx = overworldMapCell(inner, max(sb.Rows, 1))
	centerCol := sb.EastIndex + sb.Cols/2
	if hasPlayer {
		centerCol = sb.EastIndex + consts.Chunk(int(playerTile.X)/int(sb.ChunkW))
	}
	area := overworld.PlayerCenteredRange(centerCol, sb.Rows, overworldMapRadius(inner, st.cellPx))
	st.view = overworld.BuildMacroView(
		sb.RunSeed, sb.EastIndex, sb.ChunkW, sb.ChunkH,
		area, playerTile, hasPlayer, query.DriveCubeTiles(world), query.DiscoveredChunks(world, sb),
	)

	// ヘッダ表示用の現在地の絶対チャンク座標。プレイヤーが居なければ -1 にして表示を空扱いにする
	st.playerAbs = consts.Coord[consts.Chunk]{X: -1}
	if hasPlayer {
		st.playerAbs = consts.Coord[consts.Chunk]{
			X: sb.EastIndex + consts.Chunk(int(playerTile.X)/int(sb.ChunkW)),
			Y: consts.Chunk(int(playerTile.Y) / int(sb.ChunkH)),
		}
	}
	return nil
}

// overworldMapBindings は種別俯瞰図の束縛表。開いたキーと同じ N でも閉じられる
var overworldMapBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
	{Key: ebiten.KeyN, Action: inputmapper.ActionCloseMenu},
}

// Update はキー入力で閉じるだけ。マップ表示中は時間を進めない。
func (st *OverworldMapState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := keybind.ReadInput(world, overworldMapBindings); ok && action == inputmapper.ActionCloseMenu {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	return st.ConsumeTransition(), nil
}

// Draw はモーダルのパネルを描く。他のメニュー系画面と同じく画面全体は塗らず、状態機械が
// 重ね描きする背後のダンジョンの上へパネルだけを重ねる。パネルの組み立ては UI リソースを要し
// モデル算出とは別なので、初回 Draw で1度だけ組んで以後使い回す。
func (st *OverworldMapState) Draw(world w.World, screen *ebiten.Image) error {
	if st.body == nil {
		st.body = st.buildBody(world)
	}
	st.body.Draw(uicore.NewEbitenCanvas(screen))
	return nil
}

// buildBody は俯瞰図を他メニューと同じモーダルのパネルとして組む。格子・凡例・見出し・マーカーを
// 1枚の画像へ描き、menuframe の PanelBG パネルへ収めて意匠を他モーダルへ揃える。
func (st *OverworldMapState) buildBody(world w.World) uicore.Drawable {
	res := world.Resources.UIResources
	rect := menuframe.ModalRect(world)
	// 画像は ImagePanel が画像を置く矩形と同じ寸法にする。PanelInner を両者で通し余白の値が
	// 変わってもずれないようにする
	inner := menuframe.PanelInner(rect)

	// 内容はパネル内側いっぱいの画像へ原点ローカルで描く。透明地なので描かない部分は
	// 背後のパネルテクスチャが透ける
	img := ebiten.NewImage(max(inner.Dx(), 1), max(inner.Dy(), 1))
	st.renderMap(world, img)

	return menuframe.ImagePanel(res, rect, img)
}

// renderMap は俯瞰図の見出し・格子・マーカー・凡例を dst へ原点ローカルで描く。
func (st *OverworldMapState) renderMap(world w.World, dst *ebiten.Image) {
	face := world.Resources.UIResources.Text.BodyFace
	// セルの記号は小さいセルへ収めるため小フォントにする。見出し・凡例は BodyFace のまま
	glyphFace := world.Resources.UIResources.Text.SmallFace

	drawText := func(str string, x, y consts.ScreenPixel, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.ScaleWithColor(c)
		text.Draw(dst, str, face, op)
	}

	// drawCellGlyph はセルの中央に1文字を描く。基準点をセル中央に置き、水平・垂直とも中央揃えに
	// することで、字形の幅高に依らず四辺の余白が揃う
	drawCellGlyph := func(str string, cx, cy consts.ScreenPixel, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(cx), float64(cy))
		op.ColorScale.ScaleWithColor(c)
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignCenter
		text.Draw(dst, str, glyphFace, op)
	}

	drawText(fmt.Sprintf("Overworld Map  Current Chunk %d, %d", st.playerAbs.X, st.playerAbs.Y), 8, 6, theme.TextPrimary)

	cell := st.cellPx
	// 格子は横をモーダル内側の中央へ寄せる
	cols := 0
	if len(st.view.Cells) > 0 {
		cols = len(st.view.Cells[0])
	}
	gridW := consts.ScreenPixel(cols) * cell
	originX := (consts.ScreenPixel(dst.Bounds().Dx()) - gridW) / 2
	if originX < 8 {
		originX = 8
	}
	const originY consts.ScreenPixel = 40
	cellCenter := func(col, row consts.Chunk) (consts.ScreenPixel, consts.ScreenPixel) {
		x := originX + consts.ScreenPixel(col)*cell
		y := originY + consts.ScreenPixel(row)*cell
		return x + (cell-1)/2, y + (cell-1)/2
	}
	for row := range st.view.Cells {
		for col, c := range st.view.Cells[row] {
			// 未開放チャンクは描かず地を透かしてフォグにする。探索で徐々に開く
			if !c.Discovered {
				continue
			}
			r := c.Glyph
			x := originX + consts.ScreenPixel(col)*cell
			y := originY + consts.ScreenPixel(row)*cell
			// 開放済みチャンクは色を塗り、種別の文字を重ねて記号でも読めるようにする。
			// 荒れ地も含め記号は overworld が唯一の源で、UI 側で特定の記号を特別扱いしない
			vector.FillRect(dst, float32(x), float32(y), float32(cell-1), float32(cell-1), glyphColor(r), false)
			cx, cy := cellCenter(consts.Chunk(col), consts.Chunk(row))
			// 道が通るチャンクは接続方角へ線分を引く。地形塗りの上、記号の下に重ねる
			if c.Road != 0 {
				drawCellRoad(dst, x, y, cell, cx, cy, c.Road)
			}
			drawCellGlyph(string(r), cx, cy, theme.OverworldMapGlyphText)
		}
	}
	// キューブマーカー。下地は塗らず地形を残す。アイコンに暗い縁取りを付け、どの地形色でも
	// 読めるようにする。縁取りは同じアイコンを上下左右へ1pxずらして暗色で先に描く
	for _, c := range st.view.CubeCells {
		cx, cy := cellCenter(c.X, c.Y)
		for _, off := range [][2]consts.ScreenPixel{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			drawCellGlyph(consts.IconCube, cx+off[0], cy+off[1], theme.OverworldMapCubeOutline)
		}
		drawCellGlyph(consts.IconCube, cx, cy, theme.OverworldMapCubeMarker)
	}
	// 現在地マーカー。白枠でセルを囲む
	if st.view.PlayerCell.X >= 0 {
		x := originX + consts.ScreenPixel(st.view.PlayerCell.X)*cell
		y := originY + consts.ScreenPixel(st.view.PlayerCell.Y)*cell
		vector.StrokeRect(dst, float32(x-1), float32(y-1), float32(cell+1), float32(cell+1), 2, theme.OverworldMapPlayerMarker, false)
	}

	st.drawLegend(dst, drawText, drawCellGlyph, originY+consts.ScreenPixel(len(st.view.Cells))*cell+16)
}

// drawLegend は記号・色・種別名の対応を俯瞰図の下に並べて描く。色見本に格子と同じ記号を重ね、
// マップ上の1文字から凡例を引けるようにする。
func (st *OverworldMapState) drawLegend(dst *ebiten.Image, drawText func(string, consts.ScreenPixel, consts.ScreenPixel, color.Color), drawGlyph func(string, consts.ScreenPixel, consts.ScreenPixel, color.Color), top consts.ScreenPixel) {
	const swatch consts.ScreenPixel = 14
	x, y := consts.ScreenPixel(8), top
	for _, g := range overworld.LegendGlyphs() {
		vector.FillRect(dst, float32(x), float32(y), float32(swatch), float32(swatch), glyphColor(g.Label), false)
		drawGlyph(string(g.Label), x+swatch/2, y+swatch/2, theme.OverworldMapGlyphText)
		drawText(g.Name, x+20, y-2, theme.TextPrimary)
		x += 120
		if x > 720 {
			x, y = 8, y+22
		}
	}
	drawText("N / Esc to close", 8, y+26, theme.TextPrimary)
}

// drawCellRoad はチャンクセルを通る道を、接続方角ごとにセル中央から辺の中点へ細い矩形で引く。フォントに
// 罫線素片が無いので記号でなく線分で方向を見せる。各方角の矩形は中央で重なるが同色なので無害。
func drawCellRoad(dst *ebiten.Image, x, y, cell, cx, cy consts.ScreenPixel, road overworld.RoadDir) {
	t := max(consts.ScreenPixel(2), cell/5)
	half := float32(t) / 2
	left, top := float32(x), float32(y)
	right, bottom := float32(x+cell-1), float32(y+cell-1)
	cxF, cyF := float32(cx), float32(cy)
	col := theme.OverworldMapRoad
	if road&overworld.RoadW != 0 {
		vector.FillRect(dst, left, cyF-half, (cxF+half)-left, float32(t), col, false)
	}
	if road&overworld.RoadE != 0 {
		vector.FillRect(dst, cxF-half, cyF-half, right-(cxF-half), float32(t), col, false)
	}
	if road&overworld.RoadN != 0 {
		vector.FillRect(dst, cxF-half, top, float32(t), (cyF+half)-top, col, false)
	}
	if road&overworld.RoadS != 0 {
		vector.FillRect(dst, cxF-half, cyF-half, float32(t), bottom-(cyF-half), col, false)
	}
}

// glyphColor は種別文字に対応する色を返す。既知の記号は overworld の色定義を引き、
// 凡例に出ない未知の記号は灰色にする。既定色は theme に依存するのでここで決める。
func glyphColor(r rune) color.RGBA {
	if c, ok := overworld.GlyphColor(r); ok {
		return c
	}
	return theme.OverworldMapUnknownGlyph
}
