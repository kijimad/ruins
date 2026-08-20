package menuloop

import (
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// KeyHelpState は現在の文脈のキー束縛一覧を表示するステート。? でどの画面からも開ける。
// 表示は開いた画面の束縛表から導出するので、キーを変えれば一覧が追随する。
// メニュー画面では Screen の入力ゲートが ActionOpenKeyHelp を吸ってこれを push し、
// メニュー外の画面は自分の DoAction から push する
type KeyHelpState struct {
	es.BaseState[w.World]
	table  []keybind.Binding
	widget *ebitenui.UI
}

var _ es.State[w.World] = &KeyHelpState{}

// NewKeyHelpState は合成済みの表のキー一覧を表示するヘルプのファクトリを返す
func NewKeyHelpState(table []keybind.Binding) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &KeyHelpState{table: table}, nil
	}
}

// keyHelpBindings はヘルプ画面自身の束縛表。Esc か開いたときと同じ ? で閉じる
var keyHelpBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu, Label: "Back"},
	{Key: ebiten.KeySlash, Shift: keybind.ShiftRequired, Action: inputmapper.ActionCloseMenu},
}

// OnStart は一覧の UI を組む。束縛表は state の寿命の間変わらないので1度だけ組めばよい。
// キーを左寄せ、説明を右寄せの2列で揃える。キーは1粒ずつ描き、箱を持たないグリフには
// 白い箱を敷いて、全キーを白背景に黒グリフのキーキャップで統一する
func (st *KeyHelpState) OnStart(world w.World) error {
	res := world.Resources.UIResources
	entries := keybind.HintEntries(world, st.table)
	// 開いた画面の表に Esc の表示行が無ければ、ヘルプ自身の閉じ方を末尾に足す。
	// メニューは共通表の Back 行が既にあり、ダンジョンのような表には無いので補う
	if !hasEscapeLabel(st.table) {
		entries = append(entries, keybind.HintEntries(world, keyHelpBindings)...)
	}
	// NewTableContainer は幅いっぱいへ伸びる縦並びで、パネル内で行が全幅を使える
	list := styled.NewTableContainer(nil, res)
	for _, e := range entries {
		list.AddChild(keyHelpRow(e, res))
		list.AddChild(styled.NewGradientLine(res.GradientLine, theme.RowDivider, 1))
	}
	st.widget = menuframe.NewPanelScreen(res, query.T(world, "Key bindings"), list, "")
	return nil
}

// hasEscapeLabel は表に Esc の表示行があるかを返す
func hasEscapeLabel(table []keybind.Binding) bool {
	for _, b := range table {
		if b.Key == ebiten.KeyEscape && b.Label != "" {
			return true
		}
	}
	return false
}

// keyHelpRow はキー一覧の1行を組む。左にキーキャップの粒、右寄せにラベルを置く
func keyHelpRow(e keybind.HintEntry, res resources.UIResources) *widget.Container {
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(theme.Space3, 0),
			// stretch 列は preferred 幅 0 として扱われるため、キー列でなくラベル列を伸ばす。
			// キー列を伸縮にすると、パネルが preferred 幅で組まれる文脈で 0 に潰れる
			widget.GridLayoutOpts.Stretch([]bool{false, true}, []bool{false}),
			widget.GridLayoutOpts.Padding(&widget.Insets{Top: theme.Space1, Bottom: theme.Space1}),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)

	row.AddChild(widget.NewGraphic(widget.GraphicOpts.Image(renderKeycaps(e.Tokens, res))))

	label := widget.NewText(
		widget.TextOpts.Text(e.Label, &res.Text.BodyFace, theme.TextPrimary),
		widget.TextOpts.Position(widget.TextPositionEnd, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{HorizontalPosition: widget.GridLayoutPositionEnd}),
		),
	)
	row.AddChild(label)
	return row
}

// renderKeycaps はキーの粒の並びを1枚の画像へ描く。箱入りグリフはそのまま白いキーキャップに
// 見えるので素の色で描き、箱を持たないグリフには白い箱を敷いて黒で描き、見た目を揃える。
// widget の入れ子で組むと preferred 幅の計算で潰れるため、画像にして寸法を確定させる
func renderKeycaps(tokens []keybind.KeyToken, res resources.UIResources) *ebiten.Image {
	const height = 24
	const chipPad = 3
	face := res.Text.BodyFace

	type cap struct {
		text  string
		boxed bool
		w     int
	}
	caps := make([]cap, 0, len(tokens))
	total := 0
	for i, tok := range tokens {
		w, _ := text.Measure(tok.Text, face, 0)
		cw := int(w)
		if !tok.Boxed {
			cw += chipPad * 2
		}
		caps = append(caps, cap{text: tok.Text, boxed: tok.Boxed, w: cw})
		total += cw
		if i > 0 {
			total += theme.Space2
		}
	}
	if total <= 0 {
		total = 1
	}

	img := ebiten.NewImage(total, height)
	x := 0
	for i, c := range caps {
		if i > 0 {
			x += theme.Space2
		}
		clr := theme.TextPrimary
		tx := x
		if !c.boxed {
			vector.DrawFilledRect(img, float32(x), 1, float32(c.w), height-2, theme.TextPrimary, false)
			clr = theme.ScreenBackground
			tx += chipPad
		}
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(tx), 0)
		op.ColorScale.ScaleWithColor(clr)
		text.Draw(img, c.text, face, op)
		x += c.w
	}
	return img
}

// Update は閉じる入力だけを読む。ヘルプ表示中は時間を進めない
func (st *KeyHelpState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := keybind.ReadInput(world, keyHelpBindings); ok && action == inputmapper.ActionCloseMenu {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw は保持中の一覧を描く
func (st *KeyHelpState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}
