package ui

import (
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yohamta/furex/v2"
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

// Align はテキストの横方向の寄せ。
type Align int

const (
	// AlignLeft は矩形の左端に寄せる。既定。
	AlignLeft Align = iota
	// AlignRight は矩形の右端に寄せる。
	AlignRight
	// AlignCenter は矩形の中央に寄せる。
	AlignCenter
)

// Text は1行のラベル。既定は左上寄せで、Align で右寄せや中央寄せに、VCenter で縦中央にできる。
type Text struct {
	base
	Value   string
	Face    text.Face
	Color   color.Color
	Align   Align
	VCenter bool // 真なら矩形内で縦中央へ寄せる。行高が本文より高い一覧行でアイコンや強調とそろえる
}

// NewText は左上寄せのラベルを作る。
func NewText(value string, face text.Face, c color.Color) *Text {
	return &Text{Value: value, Face: face, Color: c}
}

// Layout は Text を実装する。
func (t *Text) Layout(b image.Rectangle) { t.rect = b }

// Draw は Text を実装する。Align に応じて矩形内での横位置、VCenter に応じて縦位置を決める。
// 幅・高さの測定にフェイスが要るので、フェイスが無ければ左上寄せにフォールバックする。
func (t *Text) Draw(cv Canvas) {
	x, y := t.rect.Min.X, t.rect.Min.Y
	if t.Face != nil {
		width, height := MeasureText(t.Value, t.Face)
		switch t.Align {
		case AlignLeft:
			// 左寄せは x をそのまま
		case AlignRight:
			x = t.rect.Max.X - width
		case AlignCenter:
			x = t.rect.Min.X + (t.rect.Dx()-width)/2
		}
		if t.VCenter {
			y = t.rect.Min.Y + (t.rect.Dy()-height)/2
		}
	}
	cv.DrawText(image.Pt(x, y), t.Value, t.Face, t.Color)
}

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

// Draw は Graphic を実装する。矩形に収まるよう縮小し、左寄せ・縦中央で描く。一覧のアイコンを
// 行の高さへ合わせ、左に揃えて文字と縦位置をそろえる。
func (g *Graphic) Draw(cv Canvas) {
	if g.Image != nil {
		cv.DrawImageRect(g.rect, g.Image)
	}
}

// Children は Graphic を実装する。子は持たない。
func (g *Graphic) Children() []Widget { return nil }

// NineSlice はテクスチャを9スライスで矩形いっぱいに引き伸ばして描くウィジェット。
// 枠付きの背景、窓・タイトルバー・入力枠・選択バー、に使う。BX・BY はソースの左中右・上中下の幅。
type NineSlice struct {
	base
	Image *ebiten.Image
	BX    [3]int
	BY    [3]int
}

// NewNineSlice はテクスチャとスライス幅から NineSlice を作る。
func NewNineSlice(img *ebiten.Image, bx, by [3]int) *NineSlice {
	return &NineSlice{Image: img, BX: bx, BY: by}
}

// Layout は NineSlice を実装する。
func (n *NineSlice) Layout(b image.Rectangle) { n.rect = b }

// Draw は NineSlice を実装する。
func (n *NineSlice) Draw(cv Canvas) {
	if n.Image != nil {
		cv.DrawNineSlice(n.rect, n.Image, n.BX, n.BY)
	}
}

// Children は NineSlice を実装する。子は持たない。
func (n *NineSlice) Children() []Widget { return nil }

// Group は配置済みの子をそのまま束ねて描く。子の矩形は各自が確定済みで、Group は再配置しない。
// 自前の絶対配置レイアウトを組むときに使う。
type Group struct {
	base
	children []Widget
}

// NewGroup は配置済みの子を束ねる。子は呼び出し側が Layout 済みにしておく。
func NewGroup(children ...Widget) *Group { return &Group{children: children} }

// Layout は Group を実装する。子は再配置しない。
func (g *Group) Layout(b image.Rectangle) { g.rect = b }

// Draw は Group を実装する。子を順に描く。
func (g *Group) Draw(cv Canvas) {
	for _, c := range g.children {
		c.Draw(cv)
	}
}

// Children は Group を実装する。
func (g *Group) Children() []Widget { return g.children }

// Dir はコンテナの主軸方向。
type Dir int

const (
	// Vertical は子を縦に積む。
	Vertical Dir = iota
	// Horizontal は子を横に並べる。
	Horizontal
)

// Container は子を主軸方向に並べる入れ物。背景の塗りと枠、テクスチャ背景、内側余白を持てる。
type Container struct {
	base
	dir      Dir
	sizes    []int // 主軸方向の各子のサイズ。Vertical は高さ、Horizontal は幅
	style    BoxStyle
	bgImage  *ebiten.Image // 9スライスで敷くテクスチャ背景。パネルや選択行に使う
	bgBX     [3]int
	bgBY     [3]int
	lineImg  *ebiten.Image // 非 nil なら下端に敷く区切り線のテクスチャ。横グラデを行幅へ伸ばす
	lineTint color.Color   // 区切り線の色。テクスチャに掛ける
	pad      int           // 内側余白。子はこのぶん内側へ寄せる。背景と枠は矩形いっぱいに描く
	children []Widget
}

// SetPadding は内側余白を設定する。子を余白ぶん内側へ寄せる。
func (c *Container) SetPadding(pad int) *Container {
	c.pad = pad
	return c
}

// SetStyle は背景の塗りと枠を設定する。選択行の強調などに使う。
func (c *Container) SetStyle(s BoxStyle) *Container {
	c.style = s
	return c
}

// SetBackgroundNineSlice はテクスチャ背景を9スライスで敷く。パネルや選択行の意匠に使う。
func (c *Container) SetBackgroundNineSlice(img *ebiten.Image, bx, by [3]int) *Container {
	c.bgImage = img
	c.bgBX = bx
	c.bgBY = by
	return c
}

// SetBottomLine は下端に区切り線のテクスチャを敷く。横グラデを行幅へ伸ばし tint で着色する。
// 一覧行の下線に使う。
func (c *Container) SetBottomLine(img *ebiten.Image, tint color.Color) *Container {
	c.lineImg = img
	c.lineTint = tint
	return c
}

// Layout は Container を実装する。余白を除いた内側に、furex の flexbox で子を並べる。
// 縦は sizes の固定高で積む。横は sizes の固定幅で並べ、0 幅の列すべて、1つも無ければ末尾の列が
// flex-grow で余り幅を吸収して内側を埋める。名前列を伸ばし右の数値列を右端へ寄せる用途。
// 交差軸は既定の AlignItemStretch で内側いっぱいに広がる。座標計算は furex に委譲し、
// 自前の数式を持たない。
func (c *Container) Layout(b image.Rectangle) {
	c.rect = b
	if len(c.children) == 0 {
		return
	}
	inner := image.Rect(b.Min.X+c.pad, b.Min.Y+c.pad, b.Max.X-c.pad, b.Max.Y-c.pad)

	dir := furex.Column
	fallbackIdx := -1
	if c.dir == Horizontal {
		dir = furex.Row
		hasZero := slices.Contains(c.sizes, 0)
		if !hasZero {
			// 0 幅の列が無ければ末尾の列が余り幅を吸収する。sizes と children の長さは
			// Row・VBox の構築で一致する。万一 sizes が短くても範囲外の子を伸ばして
			// 余り幅が消えないよう、幅を持つ末尾の子に上界を切る
			fallbackIdx = min(len(c.children), len(c.sizes)) - 1
		}
	}

	root := &furex.View{
		Left:      inner.Min.X,
		Top:       inner.Min.Y,
		Width:     inner.Dx(),
		Height:    inner.Dy(),
		Direction: dir,
	}
	for i, ch := range c.children {
		size := 0
		if i < len(c.sizes) {
			size = c.sizes[i]
		}
		child := &furex.View{Handler: frameTo{w: ch}}
		if c.dir == Vertical {
			child.Height = size
		} else {
			child.Width = size
			// 0 幅の列はすべて余り幅を等分して伸びる。複数あれば均等割になる
			if (size == 0 && i < len(c.sizes)) || i == fallbackIdx {
				child.Grow = 1
			}
		}
		root.AddChild(child)
	}
	root.Layout()
	root.Draw(layoutProbe())
}

// Draw は Container を実装する。テクスチャ背景、塗り、枠の順に敷いてから子を描く。
func (c *Container) Draw(cv Canvas) {
	if c.bgImage != nil {
		cv.DrawNineSlice(c.rect, c.bgImage, c.bgBX, c.bgBY)
	}
	if c.style.Fill != nil {
		cv.FillRect(c.rect, c.style.Fill)
	}
	if c.style.Border != nil && c.style.BorderWidth > 0 {
		cv.StrokeRect(c.rect, c.style.BorderWidth, c.style.Border)
	}
	for _, ch := range c.children {
		ch.Draw(cv)
	}
	if c.lineImg != nil {
		h := c.lineImg.Bounds().Dy()
		cv.DrawImageTintedRect(image.Rect(c.rect.Min.X, c.rect.Max.Y-h, c.rect.Max.X, c.rect.Max.Y), c.lineImg, c.lineTint)
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
// 先に Layout を呼ぶこと。Layout 前は各ウィジェットの矩形がゼロ値のため、ホバーは常に外れる。
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
