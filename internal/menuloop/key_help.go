package menuloop

import (
	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/ui"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// KeyHelpState は現在の文脈のキー束縛一覧を表示するステート。? でどの画面からも開ける。
// 表示は開いた画面の束縛表から導出するので、キーを変えれば一覧が追随する。
// メニュー画面では Screen の入力ゲートが ActionOpenKeyHelp を吸ってこれを push し、
// メニュー外の画面は自分の DoAction から push する
type KeyHelpState struct {
	es.BaseState[w.World]
	table []keybind.Binding
	body  ui.Widget
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
// キー列の隣に説明を左寄せで並べ、キーと説明の対応を読み取りやすくする。キーは1粒ずつ描き、
// 箱を持たないグリフには白い箱を敷いて、全キーを白背景に黒グリフのキーキャップで統一する。
// 枠・行高・余白・配置は密な一覧の標準部品 menuframe.TabScreen の既定に従う。
// 項目数が多くてもログ領域に被らず収まる
func (st *KeyHelpState) OnStart(world w.World) error {
	res := world.Resources.UIResources
	entries := keybind.HintEntries(world, st.table)
	// 開いた画面の表に Esc の表示行が無ければ、ヘルプ自身の閉じ方を末尾に足す。
	// メニューは共通表の Back 行が既にあり、ダンジョンのような表には無いので補う
	if !hasEscapeLabel(st.table) {
		entries = append(entries, keybind.HintEntries(world, keyHelpBindings)...)
	}

	// キーキャップ列の幅は最長のキーキャップ並びの実測から導く。px を直接決めず、
	// キー表記が変わっても列幅が内容へ追随する。説明列は残り幅を伸ばして左寄せにする
	caps := make([]*ebiten.Image, len(entries))
	keyColW := 0
	for i, e := range entries {
		caps[i] = renderKeycaps(e.Tokens, res)
		keyColW = max(keyColW, caps[i].Bounds().Dx())
	}
	keyColW += theme.Space3 // 説明列との間隔
	face := res.Text.BodyFace
	content := make([]ui.Widget, 0, len(entries))
	for i, e := range entries {
		label := ui.NewText(e.Label, face, theme.TextPrimary)
		label.VCenter = true
		content = append(content, ui.Row([]int{keyColW, 0}, ui.NewGraphic(caps[i]), label))
	}
	st.body = menuframe.TabScreen(world, res, query.T(world, "Key bindings"), nil, 0, content, "", "")
	return nil
}

func hasEscapeLabel(table []keybind.Binding) bool {
	for _, b := range table {
		if b.Key == ebiten.KeyEscape && b.Label != "" {
			return true
		}
	}
	return false
}

// renderKeycaps はキーの粒の並びを1枚の画像へ描く。全トークンへ一律に角丸の箱を敷き、
// 中の表記を黒で描いてキーキャップに見せる。中身は全てアイコンフォントのグリフなので、
// face 1つで描ける。
// widget の入れ子で組むと preferred 幅の計算で潰れるため、画像にして寸法を確定させる
func renderKeycaps(tokens []string, res resources.UIResources) *ebiten.Image {
	// 箱の高さは一覧アイコンの正方と同じにして、行内の見た目の粒を揃える
	const height = theme.MenuIconW
	// 箱内の左右余白とチップ間の間隔は最小間隔で統一する
	const chipPad = theme.Space2
	// 角丸の半径。キーキャップの箱の意匠
	const radius = 4

	type keycap struct {
		text string
		face text.Face
		w    int
		h    int
	}
	caps := make([]keycap, 0, len(tokens))
	total := 0
	face := res.Text.KeycapFace
	for i, tok := range tokens {
		w, h := text.Measure(tok, face, 0)
		cw := int(w) + chipPad*2
		caps = append(caps, keycap{text: tok, face: face, w: cw, h: int(h)})
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
		fillRoundedRect(img, float32(x), 0, float32(c.w), height, radius)
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(x+chipPad), float64(height-c.h)/2)
		op.ColorScale.ScaleWithColor(theme.ScreenBackground)
		text.Draw(img, c.text, c.face, op)
		x += c.w
	}
	return img
}

// fillRoundedRect は角丸の塗り矩形を描く。キーキャップの箱に使う
func fillRoundedRect(dst *ebiten.Image, x, y, w, h, r float32) {
	var p vector.Path
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(x+w, y, x+w, y+r, r)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(x+w, y+h, x+w-r, y+h, r)
	p.LineTo(x+r, y+h)
	p.ArcTo(x, y+h, x, y+h-r, r)
	p.LineTo(x, y+r)
	p.ArcTo(x, y, x+r, y, r)
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(theme.TextPrimary)
	vector.FillPath(dst, &p, nil, op)
}

// Update は閉じる入力だけを読む。ヘルプ表示中は時間を進めない
func (st *KeyHelpState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := keybind.ReadInput(world, keyHelpBindings); ok && action == inputmapper.ActionCloseMenu {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	return st.ConsumeTransition(), nil
}

// Draw は保持中の一覧を描く
func (st *KeyHelpState) Draw(_ w.World, screen *ebiten.Image) error {
	st.body.Draw(ui.NewEbitenCanvas(screen))
	return nil
}
