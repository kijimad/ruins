package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/route"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// macroMapDebugSeed はデバッグ入口でランを自動生成する際の固定シード。
const macroMapDebugSeed = 20260713

// MacroMapState はマクロ移動（ルート網のノード選択）画面のステート。
// 現在ノードの供給・寒波リードを表示し、進める辺を選んで踏破する。
// Phase 1 は簡易表示＋抽象トラベル。ノード型ごとのサブステート遷移（潜行・交易・野営）は後段。
type MacroMapState struct {
	es.BaseState[w.World]
	mount  *hooks.Mount[macroMapProps]
	widget *ebitenui.UI
}

func (st MacroMapState) String() string {
	return "MacroMap"
}

var _ es.State[w.World] = &MacroMapState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *MacroMapState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *MacroMapState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *MacroMapState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる。ラン未開始なら自動生成する（デバッグ入口用）。
func (st *MacroMapState) OnStart(world w.World) error {
	if query.GetCaravanRun(world) == nil {
		query.SetCaravanRun(world, gc.NewCaravanRun(macroMapDebugSeed, route.ExpeditionDeepVault))
	}
	st.mount = hooks.NewMount[macroMapProps]()
	return nil
}

// Update はステートの更新処理
func (st *MacroMapState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := HandleMenuInput(); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.mount.Dispatch(action)
	}

	props := st.fetchProps(world)
	st.mount.SetProps(props)
	hooks.UseTabMenu(st.mount.Store(), "macromap", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})

	if st.mount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理
func (st *MacroMapState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// DoAction はActionを実行する
func (st *MacroMapState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown,
		inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight,
		inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("macroMap: 未対応のアクション: %s", action)
	}
}

// ================
// Props
// ================

type macroMapProps struct {
	Header []string
	Items  []macroMapItem
}

type macroMapItem struct {
	Label    string
	Edge     route.Edge
	IsCancel bool // 「戻る」項目
}

func (st *MacroMapState) fetchProps(world w.World) macroMapProps {
	run := query.GetCaravanRun(world)
	cur := run.Graph.NodeByID(run.Current)

	header := []string{
		fmt.Sprintf("現在地: %s（層%d）", nodeTypeJP(cur.Type), cur.Layer),
		fmt.Sprintf("糧食 %d ／ 燃料 %d ／ 積載 %d", run.Supply.Food, run.Supply.Fuel, int(run.Supply.Cargo)),
		fmt.Sprintf("寒波リード %d 面（前進%d／前線%d）", run.FrontLead(), run.CaravanProgress, run.FrontProgress),
	}

	items := make([]macroMapItem, 0)
	for _, e := range run.Graph.Outgoing(run.Current) {
		to := run.Graph.NodeByID(e.To)
		items = append(items, macroMapItem{
			Label: fmt.Sprintf("→ %s ｜%s 面%d", nodeTypeJP(to.Type), edgeTypeJP(e.Type), e.Faces),
			Edge:  e,
		})
	}
	items = append(items, macroMapItem{Label: "戻る", IsCancel: true})

	return macroMapProps{Header: header, Items: items}
}

func (st *MacroMapState) handleSelection(world w.World) (es.Transition[w.World], error) {
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.mount, "macromap")
	if !ok {
		return es.Transition[w.World]{}, fmt.Errorf("macromapの取得に失敗")
	}
	item := st.mount.GetProps().Items[menuState.ItemIndex]
	if item.IsCancel {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	run := query.GetCaravanRun(world)
	to := run.Graph.NodeByID(item.Edge.To)
	res := run.AdvanceAlong(item.Edge)
	gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
		"%sへ移動した。糧食-%d 燃料-%d、寒波が接近する。",
		nodeTypeJP(to.Type), res.Cost.Food, res.Cost.Fuel)).Log()

	// Phase 3 は抽象トラベル。到達ノード型ごとのサブステート遷移（潜行/交易/野営）は Phase 4。
	if run.Current == run.Graph.Goal {
		gamelog.New(query.GetGameLog(world)).System("目標地点に到達した。遠征達成。").Log()
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// nodeTypeJP はノード種別の表示名を返す（表示層の関心。route はモデルを英字で持つ）。
func nodeTypeJP(t route.NodeType) string {
	switch t {
	case route.NodeHome:
		return "母港"
	case route.NodeMarket:
		return "集落マーケット"
	case route.NodeShop:
		return "専門店"
	case route.NodeRuin:
		return "遺跡"
	case route.NodeCamp:
		return "野営地"
	case route.NodeOutpost:
		return "前哨"
	case route.NodeJunction:
		return "隊商宿（合流）"
	case route.NodeGoal:
		return "目標地点"
	default:
		return "地点"
	}
}

// edgeTypeJP は辺種別の表示名を返す。
func edgeTypeJP(t route.EdgeType) string {
	switch t {
	case route.EdgeShortcut:
		return "凍える近道"
	case route.EdgeDetour:
		return "暖かい迂回"
	case route.EdgeDanger:
		return "危険路"
	default:
		return "本道"
	}
}

// ================
// buildUI
// ================

const macroMapWindowWidth = 420

func (st *MacroMapState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, "macromap")
	itemIndex := menuState.ItemIndex

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	windowContainer := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(theme.Space2),
			widget.RowLayoutOpts.Padding(&widget.Insets{Top: 20, Bottom: 20, Left: 20, Right: 20}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(macroMapWindowWidth, 0),
		),
	)

	// ヘッダ（現在地・供給・寒波リード）
	for _, line := range props.Header {
		windowContainer.AddChild(styled.NewBodyText(line, theme.TextPrimary, res))
	}

	// 進める辺の選択肢
	for i, item := range props.Items {
		isSelected := i == itemIndex
		windowContainer.AddChild(styled.NewListItemText(item.Label, theme.TextPrimary, isSelected, res))
	}

	root.AddChild(windowContainer)
	return &ebitenui.UI{Container: root}
}
