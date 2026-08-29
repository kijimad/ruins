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

	"github.com/kijimaD/ruins/internal/widgets/internal/uicore"
)

// recordCanvas は描画呼び出しを記録する Canvas。ebiten の描画コンテキスト無しで検証できる。
type recordCanvas struct {
	texts   []string
	fills   []image.Rectangle
	strokes []image.Rectangle
	images  []image.Point
}

func (c *recordCanvas) FillRect(r image.Rectangle, _ color.Color) { c.fills = append(c.fills, r) }
func (c *recordCanvas) StrokeRect(r image.Rectangle, _ int, _ color.Color) {
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

func TestSpecPanel_背景を塗り枠を描く(t *testing.T) {
	t.Parallel()
	cv := drawFixture(1)
	assert.NotEmpty(t, cv.fills, "背景の塗りが描かれる")
	assert.NotEmpty(t, cv.strokes, "枠が描かれる")
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
