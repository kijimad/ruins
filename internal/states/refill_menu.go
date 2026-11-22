package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// RefillMenuState は補充メニュー。満たすタブと抜くタブに持っている灯りを並べ、選ぶと燃料を選ぶ給油メニューを開く
type RefillMenuState struct {
	es.BaseState[w.World]
	detail overlay.Detail
	screen *menuloop.Screen[RefillProps]
}

var _ es.State[w.World] = &RefillMenuState{}
var _ menuloop.KeyBindings = &RefillMenuState{}

// NewRefillMenuState は補充メニューを作る
func NewRefillMenuState() (es.State[w.World], error) {
	return &RefillMenuState{}, nil
}

// RefillProps は補充メニューの表示 props
type RefillProps struct {
	Tabs []refillTab
}

// refillTab は補充メニューの1タブの表示。props は前の値と DeepEqual で比べて描き直すかを決めるので、値だけを持つ
type refillTab struct {
	Label string
	Rows  []refillRow
}

// refillTabSpec は補充メニューの1タブの振る舞い。タブの番号で引く。sinkOf は灯りを選んだときに開く給油メニューのくべ先を作り、
// amount は灯りの側で移せる量の上限を返す
type refillTabSpec struct {
	sinkOf func(world w.World, light ecs.Entity) FuelSink
	amount func(charge gc.Charge) int
	text   func(world w.World) refillTabText
}

// refillTabText はタブの見出しと選べない理由の文。灯りの側で移せる量が0なら lightReason、持ち物に受ける物が無ければ noneReason
type refillTabText struct {
	label       string
	lightReason string
	noneReason  string
}

var refillTabSpecs = []refillTabSpec{
	{
		sinkOf: LightFuelSink,
		amount: func(c gc.Charge) int { return c.Max - c.Current },
		text: func(world w.World) refillTabText {
			return refillTabText{label: query.T(world, "Fill"), lightReason: query.T(world, "Charge full"), noneReason: query.T(world, "No fuel")}
		},
	},
	{
		sinkOf: LightDrainSink,
		amount: func(c gc.Charge) int { return c.Current },
		text: func(world w.World) refillTabText {
			return refillTabText{label: query.T(world, "Drain"), lightReason: query.T(world, "No charge"), noneReason: query.T(world, "No container")}
		},
	},
}

// refillRow は補充メニューの1行。Disabled なら Value に選べない理由を、選べるなら移せる量を入れる。
// 選べない行もほかの物の一覧と同じ色で描き、理由の文とカーソルを止めないことで見分けさせる
type refillRow struct {
	Entity   ecs.Entity
	Value    string
	Disabled bool
}

// OnStart はステートが開始される際に呼ばれる
func (st *RefillMenuState) OnStart(_ w.World) error {
	st.detail = overlay.NewEntityDetail(st.selectedEntity)
	st.screen = menuloop.NewScreen[RefillProps](st, &st.detail)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *RefillMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *RefillMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// KeyBindings は x の詳細表示を共通入力に足す
func (st *RefillMenuState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
}

// DoAction はActionを実行する。選んだ灯りのくべ先で給油メニューを開く
func (st *RefillMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
	case inputmapper.ActionMenuSelect:
		tabIndex, row, ok := st.selected()
		if !ok || row.Disabled {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		sink := refillTabSpecs[tabIndex].sinkOf(world, row.Entity)
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) {
			return NewFeedFuelMenuState(sink)
		}}}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("refillMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// Fetch は世界から表示 props を構築する
func (st *RefillMenuState) Fetch(world w.World) (RefillProps, error) {
	lights := refillableItems(world)
	tabs := make([]refillTab, len(refillTabSpecs))
	for i, spec := range refillTabSpecs {
		text := spec.text(world)
		tabs[i].Label = text.label
		for _, light := range lights {
			tabs[i].Rows = append(tabs[i].Rows, spec.row(world, light, text))
		}
	}
	return RefillProps{Tabs: tabs}, nil
}

// row は灯りの light の1行を組む。選べなければ理由を出し、選べるなら移せる量の上限を出す
func (spec refillTabSpec) row(world w.World, light ecs.Entity, text refillTabText) refillRow {
	amount := spec.amount(*world.Components.Charge.Get(light))
	switch {
	case amount <= 0:
		return refillRow{Entity: light, Value: text.lightReason, Disabled: true}
	case !carriesAccepted(world, spec.sinkOf(world, light)):
		return refillRow{Entity: light, Value: text.noneReason, Disabled: true}
	}
	return refillRow{Entity: light, Value: consts.Turn(amount).String()}
}

// Menu はタブごとの一覧の構成を返す。選べない行はカーソルを飛ばす
func (st *RefillMenuState) Menu(props RefillProps) menuloop.MenuConfig {
	counts := make([]int, len(props.Tabs))
	skips := make([][]bool, len(props.Tabs))
	for i, tab := range props.Tabs {
		counts[i] = len(tab.Rows)
		skips[i] = make([]bool, len(tab.Rows))
		for j, row := range tab.Rows {
			skips[i][j] = row.Disabled
		}
	}
	return menuloop.MenuConfig{Key: "refill", TabCount: len(props.Tabs), ItemCounts: counts, ItemsPerPage: menuloop.ItemsPerPageAuto, Skips: skips}
}

// ViewUI はタブと、アイコンとゲージ付きの灯りの一覧を組む
func (st *RefillMenuState) ViewUI(world w.World, props RefillProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	var content []uicore.Drawable
	var pager string
	if cursor.TabIndex < len(props.Tabs) {
		tab := props.Tabs[cursor.TabIndex]
		rows := make([]menuframe.Row, len(tab.Rows))
		for i, r := range tab.Rows {
			rows[i] = itemMenuRow(world, r.Entity, 1, r.Value)
		}
		content, pager = menuframe.RenderList(cursor.ItemIndex, rows, itemMenuColumns(styled.Num()), menuframe.ListOpts{EmptyText: query.T(world, "Nothing to refill"), ItemsPerPage: cursor.PageSize}, res)
	}
	return menuframe.TabScreen(world, res, "", labels, cursor.TabIndex, content, keybind.HelpHint(world), pager)
}

// selected は現在カーソルが当たっているタブの番号と行を返す
func (st *RefillMenuState) selected() (int, refillRow, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return 0, refillRow{}, false
	}
	rows := props.Tabs[cursor.TabIndex].Rows
	if cursor.ItemIndex < 0 || cursor.ItemIndex >= len(rows) {
		return 0, refillRow{}, false
	}
	return cursor.TabIndex, rows[cursor.ItemIndex], true
}

// selectedEntity は現在カーソルが当たっている灯りを返す
func (st *RefillMenuState) selectedEntity() (ecs.Entity, bool) {
	_, row, ok := st.selected()
	return row.Entity, ok
}

// carriesAccepted はプレイヤーの持ち物に sink が受ける物があるかを返す
func carriesAccepted(world w.World, sink FuelSink) bool {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return false
	}
	for _, stack := range query.BackpackStacks(world, player) {
		if sink.accepts(world, stack.Rep) {
			return true
		}
	}
	return false
}

// refillableItems はプレイヤーが持つ物のうち、燃料を受ける灯りを満タンの物も含めて返す
func refillableItems(world w.World) []ecs.Entity {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil
	}
	var items []ecs.Entity
	q := query.ActiveFilter1[gc.Charge](world).Query()
	for q.Next() {
		e := q.Entity()
		if !query.CarriedBy(world, e, player) {
			continue
		}
		if _, ok := query.AcceptedFuelOf(world, e); ok {
			items = append(items, e)
		}
	}
	return query.SortEntities(world, items)
}
