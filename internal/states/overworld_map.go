package states

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// OverworldMapState はオーバーワールドの種別俯瞰図を全画面で表示するステート。
// 1チャンク=1建物の縮尺に合わせ、各チャンクを1マスの色と文字で示す。1マス=1つの場所。
// 生成は純関数なので ECS のタイルを生成せずに算出して描ける。
type OverworldMapState struct {
	es.BaseState[w.World]

	glyphs    [][]rune                     // 各チャンクの種別文字。行 = Y、列 = 東西の窓
	playerCol consts.Chunk                 // 現在地の窓ローカル列。範囲外なら -1
	playerRow consts.Chunk                 // 現在地の窓ローカル行
	playerAbs consts.Coord[consts.Chunk]   // 現在地の絶対チャンク座標
	cubeCells []consts.Coord[consts.Chunk] // 押せるキューブの窓ローカル (列, 行)。チャンク粒度
}

var _ es.State[w.World] = &OverworldMapState{}

// OnPause はステートが一時停止される際に呼ばれる。
func (st *OverworldMapState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる。
func (st *OverworldMapState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる。
func (st *OverworldMapState) OnStop(_ w.World) error { return nil }

// マップ描画の寸法。1チャンクを1セルで描く。
const (
	mapCellPx   consts.ScreenPixel = 22 // 1チャンクのセルの一辺ピクセル
	marginChunk consts.Chunk       = 6  // 帯の東西に足す余白チャンク数。この先の地形を先読みできる
)

// OnStart は現在地周辺の各チャンクの種別を算出して保持する。表示中はプレイヤーが動かないため
// 一度だけ計算する。
func (st *OverworldMapState) OnStart(world w.World) error {
	sb := query.GetSeamlessBand(world)
	if sb == nil || !sb.Active {
		return fmt.Errorf("overworld band is not valid")
	}
	rows := max(sb.Rows, 1)

	// 帯の Rows 行と、Cols 列に東西の余白チャンクを足した窓の各チャンクを種別文字にする
	winChunksX := int(sb.Cols + 2*marginChunk)
	winX0 := sb.EastIndex - marginChunk

	st.glyphs = make([][]rune, rows)
	for cy := range rows {
		st.glyphs[cy] = make([]rune, winChunksX)
		for i := range winChunksX {
			c := consts.Coord[consts.Chunk]{X: winX0 + consts.Chunk(i), Y: cy}
			st.glyphs[cy][i] = overworld.ChunkPlace(sb.RunSeed, c, rows)
		}
	}

	// 現在地のチャンク。プレイヤー座標は帯ローカルで、窓の原点は西へ marginChunk
	st.playerCol = -1
	if player, err := query.GetPlayerEntity(world); err == nil && world.Components.GridElement.Has(player) {
		g := world.Components.GridElement.Get(player)
		// プレイヤーの帯ローカルなチャンク座標。窓ローカル列は西へ marginChunk ぶんずらす
		localCol := consts.Chunk(int(g.X) / int(sb.ChunkW))
		localRow := consts.Chunk(int(g.Y) / int(sb.ChunkH))
		st.playerCol = localCol + marginChunk
		st.playerRow = localRow
		st.playerAbs = consts.Coord[consts.Chunk]{X: sb.EastIndex + localCol, Y: localRow}
	}

	// 押せるキューブのチャンク位置。窓の中に入るものだけを保持する。反復は最後まで回す
	st.cubeCells = nil
	cubeQuery := query.ActiveFilter2[gc.GridElement, gc.Pushable](world).Query()
	for cubeQuery.Next() {
		g := world.Components.GridElement.Get(cubeQuery.Entity())
		col := consts.Chunk(int(g.X)/int(sb.ChunkW)) + marginChunk
		row := consts.Chunk(int(g.Y) / int(sb.ChunkH))
		if col >= 0 && int(col) < winChunksX && row >= 0 && row < rows {
			st.cubeCells = append(st.cubeCells, consts.Coord[consts.Chunk]{X: col, Y: row})
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

// Draw は各チャンクを色と文字のセルで描き、現在地と凡例を添える。
func (st *OverworldMapState) Draw(world w.World, screen *ebiten.Image) error {
	screen.Fill(theme.OverworldMapBackground)
	face := world.Resources.UIResources.Text.BodyFace

	drawText := func(str string, x, y consts.ScreenPixel, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.ScaleWithColor(c)
		text.Draw(screen, str, face, op)
	}

	// drawCellGlyph はセルの中央に1文字を描く。基準点をセル中央に置き、水平・垂直とも中央揃えに
	// することで、字形の幅高に依らず四辺の余白が揃う
	drawCellGlyph := func(str string, cx, cy consts.ScreenPixel, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(cx), float64(cy))
		op.ColorScale.ScaleWithColor(c)
		op.PrimaryAlign = text.AlignCenter
		op.SecondaryAlign = text.AlignCenter
		text.Draw(screen, str, face, op)
	}

	drawText(fmt.Sprintf("Overworld Map  Current Chunk %d, %d", st.playerAbs.X, st.playerAbs.Y), 16, 12, theme.TextPrimary)

	const originX, originY consts.ScreenPixel = 16, 44
	// cellCenter はセル (col,row) の中央座標を返す。セルの塗りは一辺 mapCellPx-1
	cellCenter := func(col, row consts.Chunk) (consts.ScreenPixel, consts.ScreenPixel) {
		x := originX + consts.ScreenPixel(col)*mapCellPx
		y := originY + consts.ScreenPixel(row)*mapCellPx
		return x + (mapCellPx-1)/2, y + (mapCellPx-1)/2
	}
	for row := range st.glyphs {
		for col, r := range st.glyphs[row] {
			x := originX + consts.ScreenPixel(col)*mapCellPx
			y := originY + consts.ScreenPixel(row)*mapCellPx
			// 全チャンクを同一に扱う。色を塗り、種別の文字を重ねて記号でも読めるようにする。
			// 荒れ地も含め記号は overworld が唯一の源で、UI 側で特定の記号を特別扱いしない
			vector.FillRect(screen, float32(x), float32(y), float32(mapCellPx-1), float32(mapCellPx-1), glyphColor(r), false)
			cx, cy := cellCenter(consts.Chunk(col), consts.Chunk(row))
			drawCellGlyph(string(r), cx, cy, theme.OverworldMapGlyphText)
		}
	}
	// キューブマーカー。下地は塗らず地形を残す。アイコンに暗い縁取りを付け、どの地形色でも
	// 読めるようにする。縁取りは同じアイコンを上下左右へ1pxずらして暗色で先に描く
	for _, c := range st.cubeCells {
		cx, cy := cellCenter(c.X, c.Y)
		for _, off := range [][2]consts.ScreenPixel{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			drawCellGlyph(consts.IconCube, cx+off[0], cy+off[1], theme.OverworldMapCubeOutline)
		}
		drawCellGlyph(consts.IconCube, cx, cy, theme.OverworldMapCubeMarker)
	}
	// 現在地マーカー。白枠でセルを囲む
	if st.playerCol >= 0 {
		x := originX + consts.ScreenPixel(st.playerCol)*mapCellPx
		y := originY + consts.ScreenPixel(st.playerRow)*mapCellPx
		vector.StrokeRect(screen, float32(x-1), float32(y-1), float32(mapCellPx+1), float32(mapCellPx+1), 2, theme.OverworldMapPlayerMarker, false)
	}

	st.drawLegend(screen, drawText, drawCellGlyph, originY+consts.ScreenPixel(len(st.glyphs))*mapCellPx+16)
	return nil
}

// drawLegend は記号・色・種別名の対応を俯瞰図の下に並べて描く。色見本に格子と同じ記号を重ね、
// マップ上の1文字から凡例を引けるようにする。
func (st *OverworldMapState) drawLegend(screen *ebiten.Image, drawText func(string, consts.ScreenPixel, consts.ScreenPixel, color.Color), drawGlyph func(string, consts.ScreenPixel, consts.ScreenPixel, color.Color), top consts.ScreenPixel) {
	const swatch consts.ScreenPixel = 14
	x, y := consts.ScreenPixel(16), top
	for _, g := range overworld.LegendGlyphs() {
		vector.FillRect(screen, float32(x), float32(y), float32(swatch), float32(swatch), glyphColor(g.Label), false)
		drawGlyph(string(g.Label), x+swatch/2, y+swatch/2, theme.OverworldMapGlyphText)
		drawText(g.Name, x+20, y-2, theme.TextPrimary)
		x += 120
		if x > 720 {
			x, y = 16, y+22
		}
	}
	drawText("N / Esc to close", 16, y+26, theme.TextPrimary)
}

// glyphColorTable は文字から色への対応。色は overworld の GlyphInfo が記号と同居して持つので、
// 記号定義から一度だけ引き写す。states 側で記号ごとの色を別に定義しない。記号を変えても色定義と
// 同じ1レコードなのでずれない。凡例に出ない記号は表に入らず glyphColor の既定へ落ちる。
var glyphColorTable = func() map[rune]color.RGBA {
	table := map[rune]color.RGBA{}
	for _, g := range overworld.LegendGlyphs() {
		table[g.Label] = g.Color
	}
	return table
}()

// glyphColor は種別文字に対応する色を返す。未知の文字は灰色にする。
func glyphColor(r rune) color.RGBA {
	if c, ok := glyphColorTable[r]; ok {
		return c
	}
	return theme.OverworldMapUnknownGlyph
}
