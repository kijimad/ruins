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

	view      overworld.MacroView        // 帯全体のチャンク俯瞰。glyph 格子とプレイヤー・キューブのセル
	playerAbs consts.Coord[consts.Chunk] // 現在地の絶対チャンク座標。ヘッダ表示に使う
	body      uicore.Drawable            // モーダルのパネル。初回 Draw で1度組み以後描く
}

var _ es.State[w.World] = &OverworldMapState{}

// OnPause はステートが一時停止される際に呼ばれる。
func (st *OverworldMapState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる。
func (st *OverworldMapState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる。
func (st *OverworldMapState) OnStop(_ w.World) error { return nil }

// mapCellPx は全画面図で1チャンクを描くセルの一辺ピクセル。1チャンクを1セルで描く。
const mapCellPx consts.ScreenPixel = 22

// OnStart は帯全体の地形俯瞰モデルを算出して保持する。表示中はプレイヤーが動かないため一度だけ
// 計算する。窓構築は overworld.FullBandWindow、モデル生成は overworld.BuildMacroView に委ね、
// 右上の HUD 地図と同じ窓・同じモデルで描く。
func (st *OverworldMapState) OnStart(world w.World) error {
	sb := query.GetSeamlessBand(world)
	if sb == nil || !sb.Active {
		return fmt.Errorf("overworld band is not valid")
	}
	playerTile, hasPlayer := query.PlayerBandTile(world)
	// 全画面の俯瞰図は帯全体を窓にするが、フォグは HUD と同じで探索済みチャンクだけを開放する
	st.view = overworld.BuildMacroView(
		sb.RunSeed, sb.EastIndex, sb.ChunkW, sb.ChunkH,
		overworld.FullBandWindow(sb.EastIndex, sb.Cols, sb.Rows),
		playerTile, hasPlayer, query.DriveCubeTiles(world), query.DiscoveredChunks(world, sb),
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

// buildBody は俯瞰図を他メニューと同じモーダルのパネルとして組む。格子・凡例・見出しを1枚の
// 画像へ描き、menuframe の PanelBG パネルへ収めて意匠を他モーダルへ揃える。
func (st *OverworldMapState) buildBody(world w.World) uicore.Drawable {
	res := world.Resources.UIResources
	rect := menuframe.ModalRect(world)
	inner := image.Rect(rect.Min.X+theme.MenuPad, rect.Min.Y+theme.MenuPad, rect.Max.X-theme.MenuPad, rect.Max.Y-theme.MenuPad)

	// 内容はパネル内側いっぱいの画像へ原点ローカルで描く。透明地なので描かない部分は
	// 背後のパネルテクスチャが透ける
	img := ebiten.NewImage(max(inner.Dx(), 1), max(inner.Dy(), 1))
	st.renderMap(world, img)

	return menuframe.ImagePanel(res, rect, img)
}

// renderMap は俯瞰図の見出し・格子・マーカー・凡例を dst へ原点ローカルで描く。描画ロジックは
// 従来と同じで、描き先が screen からパネル内側の画像へ変わっただけ。
func (st *OverworldMapState) renderMap(world w.World, dst *ebiten.Image) {
	face := world.Resources.UIResources.Text.BodyFace

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
		text.Draw(dst, str, face, op)
	}

	drawText(fmt.Sprintf("Overworld Map  Current Chunk %d, %d", st.playerAbs.X, st.playerAbs.Y), 8, 6, theme.TextPrimary)

	const originX, originY consts.ScreenPixel = 8, 32
	// cellCenter はセル (col,row) の中央座標を返す。セルの塗りは一辺 mapCellPx-1
	cellCenter := func(col, row consts.Chunk) (consts.ScreenPixel, consts.ScreenPixel) {
		x := originX + consts.ScreenPixel(col)*mapCellPx
		y := originY + consts.ScreenPixel(row)*mapCellPx
		return x + (mapCellPx-1)/2, y + (mapCellPx-1)/2
	}
	for row := range st.view.Cells {
		for col, cell := range st.view.Cells[row] {
			// 未開放チャンクは描かず地を透かしてフォグにする。探索で徐々に開く
			if !cell.Discovered {
				continue
			}
			r := cell.Glyph
			x := originX + consts.ScreenPixel(col)*mapCellPx
			y := originY + consts.ScreenPixel(row)*mapCellPx
			// 開放済みチャンクは色を塗り、種別の文字を重ねて記号でも読めるようにする。
			// 荒れ地も含め記号は overworld が唯一の源で、UI 側で特定の記号を特別扱いしない
			vector.FillRect(dst, float32(x), float32(y), float32(mapCellPx-1), float32(mapCellPx-1), glyphColor(r), false)
			cx, cy := cellCenter(consts.Chunk(col), consts.Chunk(row))
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
		x := originX + consts.ScreenPixel(st.view.PlayerCell.X)*mapCellPx
		y := originY + consts.ScreenPixel(st.view.PlayerCell.Y)*mapCellPx
		vector.StrokeRect(dst, float32(x-1), float32(y-1), float32(mapCellPx+1), float32(mapCellPx+1), 2, theme.OverworldMapPlayerMarker, false)
	}

	st.drawLegend(dst, drawText, drawCellGlyph, originY+consts.ScreenPixel(len(st.view.Cells))*mapCellPx+16)
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

// glyphColor は種別文字に対応する色を返す。既知の記号は overworld の色定義を引き、
// 凡例に出ない未知の記号は灰色にする。既定色は theme に依存するのでここで決める。
func glyphColor(r rune) color.RGBA {
	if c, ok := overworld.GlyphColor(r); ok {
		return c
	}
	return theme.OverworldMapUnknownGlyph
}
