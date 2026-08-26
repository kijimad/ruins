package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Widget は保持型 UI ツリーの要素。
type Widget interface {
	// Layout は与えられた矩形内で自身と子の矩形を確定する。
	Layout(bounds image.Rectangle)
	// Draw は確定済みの矩形で Canvas に描く。
	Draw(cv Canvas)
	// Children は子を返す。ラベル収集やヒットテストに使う。
	Children() []Widget
	// Bounds は確定済みの矩形を返す。
	Bounds() image.Rectangle
}

// base は矩形を保持する共通部。各ウィジェットが埋め込む。
type base struct {
	rect image.Rectangle
}

// Bounds は確定済みの矩形を返す。
func (b *base) Bounds() image.Rectangle { return b.rect }

// Text は1行のラベル。
type Text struct {
	base
	Value string
	Face  text.Face
	Color color.Color
}

// NewText はラベルを作る。
func NewText(value string, face text.Face, c color.Color) *Text {
	return &Text{Value: value, Face: face, Color: c}
}

// Layout は Text を実装する。
func (t *Text) Layout(b image.Rectangle) { t.rect = b }

// Draw は Text を実装する。
func (t *Text) Draw(cv Canvas) { cv.DrawText(t.rect.Min, t.Value, t.Face, t.Color) }

// Children は Text を実装する。子は持たない。
func (t *Text) Children() []Widget { return nil }

// Graphic は画像を1枚描くウィジェット。
type Graphic struct {
	base
	Image *ebiten.Image
}

// NewGraphic は画像ウィジェットを作る。
func NewGraphic(img *ebiten.Image) *Graphic { return &Graphic{Image: img} }

// Layout は Graphic を実装する。
func (g *Graphic) Layout(b image.Rectangle) { g.rect = b }

// Draw は Graphic を実装する。
func (g *Graphic) Draw(cv Canvas) {
	if g.Image != nil {
		cv.DrawImage(g.rect.Min, g.Image)
	}
}

// Children は Graphic を実装する。子は持たない。
func (g *Graphic) Children() []Widget { return nil }

// Dir はコンテナの主軸方向。
type Dir int

const (
	// Vertical は子を縦に積む。
	Vertical Dir = iota
	// Horizontal は子を横に並べる。
	Horizontal
)

// Container は子を主軸方向に並べる入れ物。背景の塗りと枠を持てる。
type Container struct {
	base
	dir      Dir
	sizes    []int // 主軸方向の各子のサイズ。Vertical は高さ、Horizontal は幅
	style    BoxStyle
	children []Widget
}

// Layout は Container を実装する。主軸方向へ sizes ぶんずつ子を並べ、交差軸は矩形いっぱいに広げる。
func (c *Container) Layout(b image.Rectangle) {
	c.rect = b
	pos := b.Min
	for i, ch := range c.children {
		size := 0
		if i < len(c.sizes) {
			size = c.sizes[i]
		}
		var cell image.Rectangle
		if c.dir == Vertical {
			cell = image.Rect(b.Min.X, pos.Y, b.Max.X, pos.Y+size)
			pos.Y += size
		} else {
			cell = image.Rect(pos.X, b.Min.Y, pos.X+size, b.Max.Y)
			pos.X += size
		}
		ch.Layout(cell)
	}
}

// Draw は Container を実装する。背景を描いてから子を描く。
func (c *Container) Draw(cv Canvas) {
	if c.style.Fill != nil {
		cv.FillRect(c.rect, c.style.Fill)
	}
	if c.style.Border != nil && c.style.BorderWidth > 0 {
		cv.StrokeRect(c.rect, c.style.BorderWidth, c.style.Border)
	}
	for _, ch := range c.children {
		ch.Draw(cv)
	}
}

// Children は Container を実装する。
func (c *Container) Children() []Widget { return c.children }

// ---- 宣言的コンストラクタ ----
// ツリーを式として組む。状態が変わったら画面を組み直す。可変グローバルには触れない。

// VBox は子を縦に積む。rowH は各行の高さ。
func VBox(rowH int, children ...Widget) *Container {
	sizes := make([]int, len(children))
	for i := range sizes {
		sizes[i] = rowH
	}
	return &Container{dir: Vertical, sizes: sizes, children: children}
}

// Row は子を横に並べる。colWidths は各列の幅。
func Row(colWidths []int, cells ...Widget) *Container {
	return &Container{dir: Horizontal, sizes: colWidths, children: cells}
}

// Panel は背景付きの縦積み。rowH は各行の高さ。
func Panel(style BoxStyle, rowH int, children ...Widget) *Container {
	c := VBox(rowH, children...)
	c.style = style
	return c
}

// Input は1フレームの生入力。UI.Update に渡す。
type Input struct {
	CursorX, CursorY int
}

// UI は1つの画面。全状態を所有する。パッケージグローバルには一切置かない。
type UI struct {
	root    Widget
	cursor  image.Point
	hovered Widget
}

// New はルートを与えて UI を作る。
func New(root Widget) *UI { return &UI{root: root} }

// Root はルートウィジェットを返す。
func (u *UI) Root() Widget { return u.root }

// Layout はルートに画面矩形を与える。
func (u *UI) Layout(screen image.Rectangle) { u.root.Layout(screen) }

// Update は入力を UI 自身の状態へ取り込む。パッケージグローバルには触れないので、
// 別インスタンスの UI を並行に Update しても競合しない。
func (u *UI) Update(in Input) {
	u.cursor = image.Pt(in.CursorX, in.CursorY)
	u.hovered = hitTest(u.root, u.cursor)
}

// Draw はルートを描く。
func (u *UI) Draw(cv Canvas) { u.root.Draw(cv) }

// Hovered は現在ホバー中のウィジェットを返す。UI ごとに独立する。
func (u *UI) Hovered() Widget { return u.hovered }

// hitTest は点を含む最も内側のウィジェットを返す。含むものが無ければ nil。
func hitTest(w Widget, p image.Point) Widget {
	if !p.In(w.Bounds()) {
		return nil
	}
	for _, ch := range w.Children() {
		if hit := hitTest(ch, p); hit != nil {
			return hit
		}
	}
	return w
}

// CollectLabels はツリー内の Text.Value を出現順に集める。
func CollectLabels(w Widget) []string {
	var out []string
	if t, ok := w.(*Text); ok {
		out = append(out, t.Value)
	}
	for _, ch := range w.Children() {
		out = append(out, CollectLabels(ch)...)
	}
	return out
}
