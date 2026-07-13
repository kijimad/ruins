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
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/route"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
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

// OnStart はステートが開始される際に呼ばれる。ラン未開始なら最小の実ラン世界を用意する（デバッグ入口用）。
func (st *MacroMapState) OnStart(world w.World) error {
	// 正式な遠征選択入口（Phase 5）ではそちらがプレイヤー・ランを用意するため、ここは通らない。
	if query.GetCaravanRun(world) == nil {
		if err := setupDebugRun(world); err != nil {
			return err
		}
	}
	st.mount = hooks.NewMount[macroMapProps]()
	return nil
}

// setupDebugRun はデバッグ入口用に最小の実ラン世界（プレイヤー・隊員・ラン）を用意する。
// Shop/Dungeon などプレイヤー依存のサブステートを MacroMap から起動できるようにするための暫定処理で、
// 正式な遠征選択入口（Phase 5）ができたら不要になる。DemoStart のセットアップに倣う。
func setupDebugRun(world w.World) error {
	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	if err != nil {
		return fmt.Errorf("プレイヤーの生成に失敗: %w", err)
	}
	professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
	if len(professions) > 0 {
		if err := gameaction.ApplyProfession(world, player, professions[0]); err != nil {
			return fmt.Errorf("職業の適用に失敗: %w", err)
		}
	}
	if _, err := lifecycle.SpawnDefaultSquadMember(world, player); err != nil {
		return fmt.Errorf("初期隊員の生成に失敗: %w", err)
	}
	query.SetCaravanRun(world, gc.NewCaravanRun(macroMapDebugSeed, route.ExpeditionDeepVault))
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

	outgoing := run.Graph.Outgoing(run.Current)
	items := make([]macroMapItem, 0, len(outgoing)+1)
	for _, e := range outgoing {
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

	return st.dispatchNode(world, run, to)
}

// dispatchNode は到達ノードの型に応じたサブ挙動へ振り分ける（Phase 4）。
// Market/Shop/Ruin など実ラン世界（プレイヤー・ショップ在庫）を要する遷移は Phase 4b で接続する。
func (st *MacroMapState) dispatchNode(world w.World, run *gc.CaravanRun, node *route.Node) (es.Transition[w.World], error) {
	switch node.Type {
	case route.NodeGoal:
		gamelog.New(query.GetGameLog(world)).System("目標地点に到達した。背骨を納品して遠征達成。").Log()
		query.SetCaravanRun(world, nil) // ランを終了（再入時は新規生成）
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case route.NodeCamp:
		// 野営で糧食を回復するが、休息の間に寒波前線が詰める（道草の代償）
		const campFoodRestore = 15
		run.Supply.Food += campFoodRestore
		run.Dawdle(gc.CampFrontCost)
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"野営した。糧食を %d 回復したが、寒波が %d 面詰めた。", campFoodRestore, gc.CampFrontCost)).Log()
		if run.Swallowed() {
			return st.failSwallowed(world)
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case route.NodeMarket, route.NodeShop, route.NodeJunction, route.NodeOutpost:
		// 交易（購入・能動売却）。既存 ShopMenuState を Push し、閉じると MacroMap に戻る
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"%sに到着した。交易ができる。", nodeTypeJP(node.Type))).Log()
		return es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{NewShopMenuState},
		}, nil
	case route.NodeRuin:
		// 潜行する間に寒波前線が詰める（引き際の核）。呑まれたら潜らずラン失敗。
		// 脱出時は自動精算を通さず MacroMap を再構築して荷を持ったまま道中へ戻す（WithEscapeTarget）
		run.Dawdle(gc.RuinFrontCost)
		if run.Swallowed() {
			return st.failSwallowed(world)
		}
		gamelog.New(query.GetGameLog(world)).System(fmt.Sprintf(
			"遺跡に到着した。潜行する（寒波が %d 面詰める）。", gc.RuinFrontCost)).Log()
		return es.Transition[w.World]{
			Type: es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{
				NewDungeonState(1, WithEscapeTarget(NewMacroMapState)),
			},
		}, nil
	case route.NodeHome:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// failSwallowed は寒波前線に呑まれてラン失敗した際の処理。ランを終了して道中を閉じる。
func (st *MacroMapState) failSwallowed(world w.World) (es.Transition[w.World], error) {
	gamelog.New(query.GetGameLog(world)).System("寒波前線に呑まれた。ラン失敗。").Log()
	query.SetCaravanRun(world, nil) // ランを終了（再入時は新規生成）
	return es.Transition[w.World]{Type: es.TransPop}, nil
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
