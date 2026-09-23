package uicore_test

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// recordCanvas は描画呼び出しを記録する Canvas。ebiten の描画コンテキスト無しで検証できる。
type recordCanvas struct {
	texts          []string
	fills          []image.Rectangle
	strokes        []image.Rectangle
	roundedFills   []image.Rectangle
	roundedStrokes []image.Rectangle
	images         []image.Point
}

func (c *recordCanvas) FillRect(r image.Rectangle, _ color.Color, opts ...uicore.RectOptions) {
	if len(opts) > 0 && opts[0].Radius > 0 {
		c.roundedFills = append(c.roundedFills, r)
		return
	}
	c.fills = append(c.fills, r)
}
func (c *recordCanvas) FillTriangle(_, _, _ [2]float32, _ color.Color) {}
func (c *recordCanvas) StrokeRect(r image.Rectangle, _ int, _ color.Color, opts ...uicore.RectOptions) {
	if len(opts) > 0 && opts[0].Radius > 0 {
		c.roundedStrokes = append(c.roundedStrokes, r)
		return
	}
	c.strokes = append(c.strokes, r)
}
func (c *recordCanvas) DrawText(_ image.Point, s string, _ text.Face, _ color.Color) {
	c.texts = append(c.texts, s)
}
func (c *recordCanvas) DrawImage(p image.Point, _ *ebiten.Image) { c.images = append(c.images, p) }
func (c *recordCanvas) DrawImageRect(dst image.Rectangle, _ *ebiten.Image) {
	c.images = append(c.images, dst.Min)
}
func (c *recordCanvas) DrawImageTintedRect(dst image.Rectangle, _ *ebiten.Image, _ color.Color) {
	c.images = append(c.images, dst.Min)
}
func (c *recordCanvas) DrawNineSlice(dst image.Rectangle, _ *ebiten.Image, _, _ [3]int) {
	c.images = append(c.images, dst.Min)
}

// specRow は ruins の entityspec.SpecRow を模した表示データ。pure。
type specRow struct {
	label string
	value string
}

// buildSpecPanel は ruins の RenderSpecRows を模す。宣言的にツリーを式として組む。
// グローバルに触れないので、複数ゴルーチンから同時に呼んでも安全。
func buildSpecPanel(rows []specRow) *uicore.Container {
	cols := []int{70, 80}
	items := make([]uicore.Widget, 0, len(rows))
	for _, r := range rows {
		items = append(items, uicore.Row(cols, label(r.label), label(r.value)))
	}
	style := uicore.BoxStyle{Fill: color.Gray{Y: 20}, Border: color.White, BorderWidth: 1}
	return uicore.Panel(style, 16, items...)
}

// label は既定のフェイス無しでラベルを作る。fake canvas はフェイスを無視する。
func label(s string) *uicore.Text { return uicore.NewText(s, nil, color.White) }

// drawFixture は i 番目のパネルを組んで描画し、記録を返す。
func drawFixture(i int) *recordCanvas {
	panel := buildSpecPanel(specFixture(i))
	panel.Layout(image.Rect(0, 0, 300, 400))
	cv := &recordCanvas{}
	panel.Draw(cv)
	return cv
}

func specFixture(i int) []specRow {
	return []specRow{
		{"Vitality", fmt.Sprintf("%d", 10+i)},
		{"Strength", fmt.Sprintf("%d", 11+i)},
		{"Agility", fmt.Sprintf("%d", 14+i)},
		{"Defense", fmt.Sprintf("%d", 15+i)},
	}
}

func TestSpecPanel_能力値を表示する(t *testing.T) {
	t.Parallel()
	cv := drawFixture(0)
	assert.Contains(t, cv.texts, "Vitality", "体力ラベルが表示される")
	assert.Contains(t, cv.texts, "10", "体力の値が表示される")
	assert.Contains(t, cv.texts, "Defense", "防御ラベルが表示される")
	assert.Contains(t, cv.texts, "15", "防御の値が表示される")
}

func TestSpecPanel_背景をパネル全体に1つ塗り枠を描く(t *testing.T) {
	t.Parallel()
	cv := drawFixture(1)
	// 背景はパネル1枚ぶんだけ。行は個別の背景を持たない
	assert.Equal(t, []image.Rectangle{image.Rect(0, 0, 300, 400)}, cv.fills, "塗りはパネル全体を1つだけ覆う")
	assert.Equal(t, []image.Rectangle{image.Rect(0, 0, 300, 400)}, cv.strokes, "枠もパネル全体を1つだけ描く")
}

func TestSpecPanel_行数ぶんのラベルが出る(t *testing.T) {
	t.Parallel()
	panel := buildSpecPanel(specFixture(2))
	panel.Layout(image.Rect(0, 0, 300, 400))
	assert.Len(t, uicore.CollectLabels(panel), 8, "4行 かける ラベルと値の2列")
}

func TestSpecPanel_異なる入力は独立する(t *testing.T) {
	t.Parallel()
	a := drawFixture(0)
	b := drawFixture(100)
	assert.Contains(t, a.texts, "10")
	assert.Contains(t, b.texts, "110")
	assert.NotContains(t, a.texts, "110", "別インスタンスの値が混ざらない")
}

func TestSpecPanel_多数の並列構築(t *testing.T) {
	t.Parallel()
	for i := range 20 {
		t.Run(fmt.Sprintf("i=%d", i), func(t *testing.T) {
			t.Parallel()
			cv := drawFixture(i)
			assert.Contains(t, cv.texts, fmt.Sprintf("%d", 10+i))
		})
	}
}

// フレーム駆動を模す。別インスタンスのツリーを多数ゴルーチンで同時に配置・描画し続ける。
// ミューテックスでは意味が壊れたケース。インスタンス所有ならロック無しで競合しない。
func TestConcurrentLayoutDraw_インスタンスごとに独立(t *testing.T) {
	t.Parallel()
	const workers = 32
	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			panel := buildSpecPanel(specFixture(id))
			for range 50 {
				panel.Layout(image.Rect(0, 0, 300, 400))
				panel.Draw(&recordCanvas{})
			}
		}(w)
	}
	wg.Wait()
}

func TestBox_Draw_塗りと枠を角丸で描く(t *testing.T) {
	t.Parallel()
	rr := uicore.NewBox(color.Gray{Y: 30}, color.White, 7)
	rr.Layout(image.Rect(0, 0, 100, 50))
	cv := &recordCanvas{}
	rr.Draw(cv)
	assert.Len(t, cv.roundedFills, 1, "塗りを角丸で1つ描く")
	assert.Len(t, cv.roundedStrokes, 1, "枠を角丸で1つ描く")
	assert.Empty(t, cv.fills, "直角の塗りは描かない")
}

func TestBox_Draw_borderがnilなら枠を描かない(t *testing.T) {
	t.Parallel()
	rr := uicore.NewBox(color.Gray{Y: 30}, nil, 7)
	rr.Layout(image.Rect(0, 0, 100, 50))
	cv := &recordCanvas{}
	rr.Draw(cv)
	assert.Len(t, cv.roundedFills, 1)
	assert.Empty(t, cv.roundedStrokes, "border が nil なら枠を描かない")
}

func TestContainer_SetStyle_角丸背景を敷く(t *testing.T) {
	t.Parallel()
	c := uicore.Panel(uicore.BoxStyle{}, 16).SetStyle(uicore.BoxStyle{Fill: color.Gray{Y: 20}, Border: color.White, BorderWidth: 1, Radius: 7})
	c.Layout(image.Rect(0, 0, 120, 60))
	cv := &recordCanvas{}
	c.Draw(cv)
	assert.Len(t, cv.roundedFills, 1, "背景の塗りを角丸で1つ敷く")
	assert.Len(t, cv.roundedStrokes, 1, "枠を角丸で1つ敷く")
	assert.Empty(t, cv.fills, "直角の塗りは描かない")
}
