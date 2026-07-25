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
	playerCol int                        // 現在地の列。範囲外なら -1
	playerRow int                        // 現在地の行
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
	mapCellPx    = 22  // 1チャンクのセルの一辺ピクセル
	mapContextCh = 6   // 帯の東西に足す文脈チャンク数。この先の地形を先読みできる
	fieldGlyph   = '.' // 荒れ地の記号。overworld 側の featureField と同じ。荒れ地は色だけで文字を重ねない
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
	winChunksX := int(sb.K) + 2*mapContextCh
	winX0 := sb.EastIndex - consts.Chunk(mapContextCh)

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
		st.playerCol = int(g.X)/int(sb.ChunkW) + mapContextCh
		st.playerRow = int(g.Y) / int(sb.ChunkH)
		st.playerAbs = consts.Coord[consts.Chunk]{
			X: sb.EastIndex + consts.Chunk(int(g.X)/int(sb.ChunkW)),
			Y: consts.Chunk(int(g.Y) / int(sb.ChunkH)),
		}
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
	screen.Fill(color.RGBA{R: 12, G: 14, B: 18, A: 255})
	face := world.Resources.UIResources.Text.BodyFace

	drawText := func(str string, x, y int, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.ScaleWithColor(c)
		text.Draw(screen, str, face, op)
	}

	drawText(fmt.Sprintf("オーバーワールド地図  現在地 チャンク(%d, %d)", st.playerAbs.X, st.playerAbs.Y), 16, 12, theme.TextPrimary)

	const originX, originY = 16, 44
	for row := range st.glyphs {
		for col, r := range st.glyphs[row] {
			x := originX + col*mapCellPx
			y := originY + row*mapCellPx
			vector.FillRect(screen, float32(x), float32(y), mapCellPx-1, mapCellPx-1, glyphColor(r), false)
			// 原野以外は種別の文字を重ねて、色だけでなく記号でも読めるようにする
			if r != fieldGlyph {
				drawText(string(r), x+5, y+2, color.RGBA{R: 20, G: 20, B: 24, A: 255})
			}
		}
	}
	// 現在地マーカー。白枠でセルを囲む
	if st.playerCol >= 0 {
		x := float32(originX + st.playerCol*mapCellPx)
		y := float32(originY + st.playerRow*mapCellPx)
		vector.StrokeRect(screen, x-1, y-1, mapCellPx+1, mapCellPx+1, 2, color.RGBA{R: 255, G: 255, B: 255, A: 255}, false)
	}

	st.drawLegend(screen, drawText, originY+len(st.glyphs)*mapCellPx+16)
	return nil
}

// drawLegend は色と種別名の対応を俯瞰図の下に並べて描く。凡例の記号と名前は overworld が
// 唯一の源として持つので、ここで種別名を直書きしない。
func (st *OverworldMapState) drawLegend(screen *ebiten.Image, drawText func(string, int, int, color.Color), top int) {
	x, y := 16, top
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

// facilityPalette は施設記号から色への対応。FacilityGlyphs() の並び順に依存させず、記号
// そのものをキーにして、施設種別を iota の途中に足しても色がずれないようにする。
var facilityPalette = map[rune]color.RGBA{
	'h': {R: 154, G: 160, B: 166, A: 255}, // 住宅 灰
	'S': {R: 74, G: 144, B: 226, A: 255},  // 商店 青
	'O': {R: 80, G: 200, B: 208, A: 255},  // 事務所 シアン
	'D': {R: 176, G: 122, B: 58, A: 255},  // 倉庫 茶
	'A': {R: 212, G: 160, B: 23, A: 255},  // 骨董品店 金
	'C': {R: 232, G: 106, B: 154, A: 255}, // 診療所 桃
	'L': {R: 160, G: 106, B: 208, A: 255}, // 研究施設 紫
}

// featurePalette は地物記号から色への対応。facilityPalette と同じく記号そのものをキーにして、
// overworld 側の featureKind の並びに依存させない。色の無い記号は glyphColor の既定へ落ちる。
var featurePalette = map[rune]color.RGBA{
	'.': {R: 46, G: 59, B: 46, A: 255},    // 荒れ地 暗緑
	'T': {R: 255, G: 210, B: 74, A: 255},  // 村 黄
	't': {R: 208, G: 168, B: 58, A: 255},  // 一軒家 濃黄
	'>': {R: 224, G: 69, B: 58, A: 255},   // 遺跡入口 赤
	'*': {R: 111, G: 191, B: 111, A: 255}, // 点在POI 緑
}

// glyphColorTable は文字から色への対応。init で地物と施設の色を1つの表にまとめる。
var glyphColorTable = map[rune]color.RGBA{}

func init() {
	// 記号キーで引くので overworld 側の並び順に依存しない。色の無い記号は glyphColor の既定へ落ちる
	for _, g := range overworld.FeatureGlyphs() {
		if c, ok := featurePalette[g.Label]; ok {
			glyphColorTable[g.Label] = c
		}
	}
	for _, g := range overworld.FacilityGlyphs() {
		if c, ok := facilityPalette[g.Label]; ok {
			glyphColorTable[g.Label] = c
		}
	}
}

// glyphColor は種別文字に対応する色を返す。未知の文字は灰色にする。
func glyphColor(r rune) color.RGBA {
	if c, ok := glyphColorTable[r]; ok {
		return c
	}
	return color.RGBA{R: 90, G: 90, B: 90, A: 255}
}
