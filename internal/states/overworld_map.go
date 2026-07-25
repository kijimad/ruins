package states

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
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

	glyphs    [][]rune                   // 各チャンクの種別文字。行 = Y、列 = 東西の窓
	playerCol consts.Chunk               // 現在地の窓ローカル列。範囲外なら -1
	playerRow consts.Chunk               // 現在地の窓ローカル行
	playerAbs consts.Coord[consts.Chunk] // 現在地の絶対チャンク座標
}

var _ es.State[w.World] = &OverworldMapState{}
var _ Configurable = &OverworldMapState{}

// StateConfig はこのステートの設定を返す。俯瞰図は自前で背景を塗るのでぼかしは使わない。
func (st *OverworldMapState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

// OnPause はステートが一時停止される際に呼ばれる。
func (st *OverworldMapState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる。
func (st *OverworldMapState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる。
func (st *OverworldMapState) OnStop(_ w.World) error { return nil }

// マップ描画の寸法。1チャンクを1セルで描く。
const (
	mapCellPx    consts.ScreenPixel = 22 // 1チャンクのセルの一辺ピクセル
	mapContextCh consts.Chunk       = 6  // 帯の東西に足す文脈チャンク数。この先の地形を先読みできる
)

// OnStart は現在地周辺の各チャンクの種別を算出して保持する。表示中はプレイヤーが動かないため
// 一度だけ計算する。
func (st *OverworldMapState) OnStart(world w.World) error {
	sb := query.GetSeamlessBand(world)
	if sb == nil || !sb.Active {
		return fmt.Errorf("オーバーワールド帯が有効でない")
	}
	rows := max(sb.Rows, 1)

	// 帯の Rows 行と、K 列に東西の文脈チャンクを足した窓の各チャンクを種別文字にする
	winChunksX := int(sb.K + 2*mapContextCh)
	winX0 := sb.EastIndex - mapContextCh

	st.glyphs = make([][]rune, rows)
	for cy := range rows {
		st.glyphs[cy] = make([]rune, winChunksX)
		for i := range winChunksX {
			c := consts.Coord[consts.Chunk]{X: winX0 + consts.Chunk(i), Y: cy}
			st.glyphs[cy][i] = overworld.ChunkPlace(sb.RunSeed, c, rows)
		}
	}

	// 現在地のチャンク。プレイヤー座標は帯ローカルで、窓の原点は西へ mapContextCh
	st.playerCol = -1
	if player, err := query.GetPlayerEntity(world); err == nil && world.Components.GridElement.Has(player) {
		g := world.Components.GridElement.Get(player)
		// プレイヤーの帯ローカルなチャンク座標。窓ローカル列は西へ mapContextCh ぶんずらす
		localCol := consts.Chunk(int(g.X) / int(sb.ChunkW))
		localRow := consts.Chunk(int(g.Y) / int(sb.ChunkH))
		st.playerCol = localCol + mapContextCh
		st.playerRow = localRow
		st.playerAbs = consts.Coord[consts.Chunk]{X: sb.EastIndex + localCol, Y: localRow}
	}
	return nil
}

// Update はキー入力で閉じるだけ。マップ表示中は時間を進めない。
func (st *OverworldMapState) Update(_ w.World) (es.Transition[w.World], error) {
	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) ||
		keyboardInput.IsKeyJustPressed(ebiten.KeyN) ||
		keyboardInput.IsKeyJustPressed(ebiten.KeyM) {
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

	drawText(fmt.Sprintf("オーバーワールド地図  現在地 チャンク(%d, %d)", st.playerAbs.X, st.playerAbs.Y), 16, 12, theme.TextPrimary)

	const originX, originY consts.ScreenPixel = 16, 44
	for row := range st.glyphs {
		for col, r := range st.glyphs[row] {
			x := originX + consts.ScreenPixel(col)*mapCellPx
			y := originY + consts.ScreenPixel(row)*mapCellPx
			// 全チャンクを同一に扱う。色を塗り、種別の文字を重ねて記号でも読めるようにする。
			// 荒れ地も含め記号は overworld が唯一の源で、UI 側で特定の記号を特別扱いしない
			vector.FillRect(screen, float32(x), float32(y), float32(mapCellPx-1), float32(mapCellPx-1), glyphColor(r), false)
			drawText(string(r), x+5, y+2, theme.OverworldMapGlyphText)
		}
	}
	// 現在地マーカー。白枠でセルを囲む
	if st.playerCol >= 0 {
		x := originX + consts.ScreenPixel(st.playerCol)*mapCellPx
		y := originY + consts.ScreenPixel(st.playerRow)*mapCellPx
		vector.StrokeRect(screen, float32(x-1), float32(y-1), float32(mapCellPx+1), float32(mapCellPx+1), 2, theme.OverworldMapPlayerMarker, false)
	}

	st.drawLegend(screen, drawText, originY+consts.ScreenPixel(len(st.glyphs))*mapCellPx+16)
	return nil
}

// drawLegend は色と種別名の対応を俯瞰図の下に並べて描く。凡例の記号と名前は overworld が
// 唯一の源として持つので、ここで種別名を直書きしない。
func (st *OverworldMapState) drawLegend(screen *ebiten.Image, drawText func(string, consts.ScreenPixel, consts.ScreenPixel, color.Color), top consts.ScreenPixel) {
	x, y := consts.ScreenPixel(16), top
	for _, g := range overworld.LegendGlyphs() {
		vector.FillRect(screen, float32(x), float32(y), 14, 14, glyphColor(g.Label), false)
		drawText(g.Name, x+20, y-2, theme.TextPrimary)
		x += 120
		if x > 720 {
			x, y = 16, y+22
		}
	}
	drawText("N / Esc で閉じる", 16, y+26, theme.TextPrimary)
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
