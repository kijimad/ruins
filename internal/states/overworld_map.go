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
	"github.com/kijimaD/ruins/internal/worldstream"
)

// OverworldMapState はオーバーワールドの種別俯瞰図を全画面で表示するステート。
// タイルの材質ではなく「そのマスがどの種別の場所か」を色で示す。帯の生成は純関数なので、
// 現在地の周辺チャンクを ECS を生成せずに算出して描ける。
type OverworldMapState struct {
	es.BaseState[w.World]

	grid      [][]rune // ダウンサンプル済みの種別文字グリッド
	playerCol int      // 現在地のグリッド列。範囲外なら -1
	playerRow int      // 現在地のグリッド行
	playerAbs consts.Coord[consts.Chunk]
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

// マップ描画の寸法。論理解像度 960x720 に収める。凡例を右端に置く余白を残す。
const (
	mapCellPx    = 6   // ダウンサンプル1マスの一辺ピクセル
	mapAreaW     = 760 // 俯瞰図領域の最大幅。右に凡例の帯を残す
	mapAreaH     = 560 // 俯瞰図領域の最大高
	mapContextCh = 2   // 帯の東西に足す文脈チャンク数。この先の地形を先読みできる
)

// OnStart は現在地周辺の俯瞰図をダウンサンプルして構築する。マップ表示中はプレイヤーが
// 動かないため、ここで一度だけ計算して保持する。
func (st *OverworldMapState) OnStart(world w.World) error {
	sb := query.GetSeamlessBand(world)
	if sb == nil || !sb.Active {
		return fmt.Errorf("オーバーワールド帯が有効でない")
	}
	rows := max(sb.Rows, 1)

	// 帯の Rows 行と、K 列に東西の文脈チャンクを足した窓を対象にする
	winChunksX := int(sb.K) + 2*mapContextCh
	winX0 := sb.EastIndex - consts.Chunk(mapContextCh)
	tilesW := consts.Tile(winChunksX) * sb.ChunkW
	tilesH := rows.Tiles(sb.ChunkH)

	// 領域に収まるダウンサンプル率を決める。行数が増えても自動で縮尺が合う
	maxCols := mapAreaW / mapCellPx
	maxRows := mapAreaH / mapCellPx
	dsample := max(1, ceilDiv(int(tilesW), maxCols), ceilDiv(int(tilesH), maxRows))

	// 窓の全タイル種別を1枚のグリッドへ展開する
	full := make([][]rune, tilesH)
	for y := range full {
		full[y] = make([]rune, tilesW)
	}
	for i := range winChunksX {
		for cy := consts.Chunk(0); cy < rows; cy++ {
			c := worldstream.ChunkCoord{X: winX0 + consts.Chunk(i), Y: cy}
			chunk := overworld.ChunkSchematic(sb.RunSeed, c, rows, sb.ChunkW, sb.ChunkH)
			baseX := consts.Tile(i) * sb.ChunkW
			baseY := cy.Tiles(sb.ChunkH)
			for y, line := range chunk {
				copy(full[int(baseY)+y][int(baseX):], []rune(line))
			}
		}
	}

	// ダウンサンプル。各ブロックの代表は原野でない最初の文字にし、地物を優先して残す
	st.grid = downsampleRunes(full, dsample)

	// 現在地のグリッド座標を求める。プレイヤー座標は帯ローカルで、窓の原点は西へ mapContextCh
	if player, err := query.GetPlayerEntity(world); err == nil && world.Components.GridElement.Has(player) {
		g := world.Components.GridElement.Get(player)
		winTileX := int(g.X) + mapContextCh*int(sb.ChunkW)
		st.playerCol = winTileX / dsample
		st.playerRow = int(g.Y) / dsample
		st.playerAbs = consts.Coord[consts.Chunk]{
			X: sb.EastIndex + consts.Chunk(int(g.X)/int(sb.ChunkW)),
			Y: consts.Chunk(int(g.Y) / int(sb.ChunkH)),
		}
	} else {
		st.playerCol = -1
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

// Draw は俯瞰図をカラーのミニマップとして描き、現在地と凡例を添える。
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

	// ミニマップ本体
	const originX, originY = 16, 44
	for row := range st.grid {
		for col, r := range st.grid[row] {
			x := float32(originX + col*mapCellPx)
			y := float32(originY + row*mapCellPx)
			vector.FillRect(screen, x, y, mapCellPx, mapCellPx, glyphColor(r), false)
		}
	}
	// 現在地マーカー
	if st.playerCol >= 0 {
		x := float32(originX + st.playerCol*mapCellPx - 1)
		y := float32(originY + st.playerRow*mapCellPx - 1)
		vector.FillRect(screen, x, y, mapCellPx+2, mapCellPx+2, color.RGBA{R: 255, G: 255, B: 255, A: 255}, false)
	}

	// 凡例
	st.drawLegend(screen, drawText)
	return nil
}

// drawLegend は色と種別名の対応を右端に縦並びで描く。
func (st *OverworldMapState) drawLegend(screen *ebiten.Image, drawText func(string, int, int, color.Color)) {
	const legendX, legendY = mapAreaW + 32, 44
	type entry struct {
		r    rune
		name string
	}
	facilities := overworld.FacilityGlyphs()
	entries := make([]entry, 0, 5+len(facilities))
	entries = append(entries,
		entry{overworld.GlyphField, "原野"},
		entry{overworld.GlyphVillage, "村"},
		entry{overworld.GlyphHamlet, "一軒家"},
		entry{overworld.GlyphRuin, "遺跡入口"},
		entry{overworld.GlyphPOI, "点在POI"},
	)
	for _, g := range facilities {
		entries = append(entries, entry{g.Label, g.Name})
	}
	y := legendY
	for _, e := range entries {
		vector.FillRect(screen, float32(legendX), float32(y), 14, 14, glyphColor(e.r), false)
		drawText(e.name, legendX+20, y-2, theme.TextPrimary)
		y += 22
	}
	vector.FillRect(screen, float32(legendX), float32(y+2), 14, 14, color.RGBA{R: 255, G: 255, B: 255, A: 255}, false)
	drawText("現在地", legendX+20, y+2, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	drawText("N / Esc で閉じる", 16, mapAreaH+52, theme.TextPrimary)
}

// facilityPalette は施設種別の色。overworld.FacilityGlyphs() の種別順に対応する。
var facilityPalette = []color.RGBA{
	{R: 154, G: 160, B: 166, A: 255}, // 住宅 灰
	{R: 74, G: 144, B: 226, A: 255},  // 商店 青
	{R: 80, G: 200, B: 208, A: 255},  // 事務所 シアン
	{R: 176, G: 122, B: 58, A: 255},  // 倉庫 茶
	{R: 212, G: 160, B: 23, A: 255},  // 骨董品店 金
	{R: 232, G: 106, B: 154, A: 255}, // 診療所 桃
	{R: 160, G: 106, B: 208, A: 255}, // 研究施設 紫
}

// glyphColorTable は文字から色への対応。init で地物と施設の色を1つの表にまとめる。
var glyphColorTable = map[rune]color.RGBA{}

func init() {
	glyphColorTable[overworld.GlyphField] = color.RGBA{R: 46, G: 59, B: 46, A: 255}
	glyphColorTable[overworld.GlyphRoad] = color.RGBA{R: 138, G: 138, B: 106, A: 255}
	glyphColorTable[overworld.GlyphVillage] = color.RGBA{R: 255, G: 210, B: 74, A: 255}
	glyphColorTable[overworld.GlyphHamlet] = color.RGBA{R: 208, G: 168, B: 58, A: 255}
	glyphColorTable[overworld.GlyphRuin] = color.RGBA{R: 224, G: 69, B: 58, A: 255}
	glyphColorTable[overworld.GlyphPOI] = color.RGBA{R: 111, G: 191, B: 111, A: 255}
	for i, g := range overworld.FacilityGlyphs() {
		if i < len(facilityPalette) {
			glyphColorTable[g.Label] = facilityPalette[i]
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

// downsampleRunes は step マスごとに、原野でない最初の文字を代表として縮約する。
// 地物や建物を原野より優先して残すので、縮尺を上げても存在が消えにくい。
func downsampleRunes(full [][]rune, step int) [][]rune {
	if step < 1 {
		step = 1
	}
	out := make([][]rune, 0, len(full)/step+1)
	for y := 0; y < len(full); y += step {
		var line []rune
		for x := 0; x < len(full[y]); x += step {
			r := overworld.GlyphField
			for dy := 0; dy < step && y+dy < len(full); dy++ {
				for dx := 0; dx < step && x+dx < len(full[y+dy]); dx++ {
					if full[y+dy][x+dx] != overworld.GlyphField && full[y+dy][x+dx] != 0 {
						r = full[y+dy][x+dx]
						break
					}
				}
				if r != overworld.GlyphField {
					break
				}
			}
			line = append(line, r)
		}
		out = append(out, line)
	}
	return out
}

// ceilDiv は切り上げ除算。b>0 を前提にする。
func ceilDiv(a, b int) int {
	if b <= 0 {
		return a
	}
	return (a + b - 1) / b
}
