package states

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// squadSubState はサブステート
type squadSubState int

const (
	squadSubStateMenu   squadSubState = iota // 隊員一覧
	squadSubStateWindow                      // アクションウィンドウ
)

// SquadMenuState は隊員管理のゲームステート
type SquadMenuState struct {
	es.BaseState[w.World]
	subState    squadSubState
	menuMount   *hooks.Mount[squadProps]
	windowMount *hooks.Mount[squadWindowProps]
	widget      *ebitenui.UI
}

func (st SquadMenuState) String() string {
	return "SquadMenu"
}

var _ es.State[w.World] = &SquadMenuState{}
var _ es.ActionHandler[w.World] = &SquadMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *SquadMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *SquadMenuState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *SquadMenuState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始する際に呼ばれる
func (st *SquadMenuState) OnStart(_ w.World) error {
	st.subState = squadSubStateMenu
	st.menuMount = hooks.NewMount[squadProps]()
	st.windowMount = hooks.NewMount[squadWindowProps]()
	return nil
}

// Update はステートの更新処理を行う
func (st *SquadMenuState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		switch st.subState {
		case squadSubStateMenu:
			st.menuMount.Dispatch(action)
		case squadSubStateWindow:
			st.windowMount.Dispatch(action)
		}
	}

	props := st.fetchProps(world)
	st.menuMount.SetProps(props)

	hooks.UseTabMenu(st.menuMount.Store(), "squad", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.BatchCommands) + len(props.Members)},
	})

	st.setupWindowState(world)

	menuDirty := st.menuMount.Update()
	windowDirty := st.windowMount.Update()
	if menuDirty || windowDirty || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理を行う
func (st *SquadMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// HandleInput は入力を処理してアクションIDを返す
func (st *SquadMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	switch st.subState {
	case squadSubStateMenu:
		return HandleMenuInput()
	case squadSubStateWindow:
		return HandleWindowInput()
	}
	return "", false
}

// DoAction はアクションを実行してステート遷移を返す
func (st *SquadMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch st.subState {
	case squadSubStateWindow:
		switch action {
		case inputmapper.ActionWindowConfirm:
			if err := st.executeWindowAction(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionWindowCancel:
			st.subState = squadSubStateMenu
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			// Dispatchで処理
		default:
			return es.Transition[w.World]{}, fmt.Errorf("squadSubStateWindow: 未対応のアクション: %s", action)
		}

	case squadSubStateMenu:
		switch action {
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			return es.Transition[w.World]{Type: es.TransPop}, nil
		case inputmapper.ActionMenuSelect:
			st.handleMemberSelection(world)
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
			// Dispatchで処理
		default:
			return es.Transition[w.World]{}, fmt.Errorf("squadSubStateMenu: 未対応のアクション: %s", action)
		}
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

type squadProps struct {
	// BatchCommands は一括操作コマンド。隊員一覧の前に表示される
	BatchCommands []string
	Members       []squadMemberData
}

type squadMemberData struct {
	Entity       ecs.Entity
	Name         string
	HP           string
	Position     string
	Combat       string
	ItemPickup   string
	ItemHandling string
}

type squadWindowProps struct {
	Member squadMemberData
}

var batchCommands = []string{"集合", "全員待機"}

func (st *SquadMenuState) fetchProps(world w.World) squadProps {
	var members []squadMemberData

	_, err := query.GetPlayerEntity(world)
	if err != nil {
		return squadProps{}
	}

	for _, member := range query.SquadMembers(world) {
		name := query.GetEntityName(member, world)
		hp := world.Components.HP.Get(member).(*gc.HP)
		ai := query.GetAI(world, member)
		if ai == nil {
			continue
		}

		members = append(members, squadMemberData{
			Entity:       member,
			Name:         name,
			HP:           fmt.Sprintf("%d/%d", hp.Current, hp.Max),
			Position:     ai.Movement.String(),
			Combat:       ai.CombatCurrent.Label(),
			ItemPickup:   ai.ItemPickup.String(),
			ItemHandling: ai.ItemHandling.String(),
		})
	}

	return squadProps{BatchCommands: batchCommands, Members: members}
}

// ================
// Window
// ================

func (st *SquadMenuState) setupWindowState(_ w.World) {
	actionItems := st.getActionItems()

	hooks.UseState(st.windowMount.Store(), "squad_window_index", 0, func(v int, action inputmapper.ActionID) int {
		switch action {
		case inputmapper.ActionWindowUp:
			if v > 0 {
				return v - 1
			}
			return len(actionItems) - 1
		case inputmapper.ActionWindowDown:
			if v < len(actionItems)-1 {
				return v + 1
			}
			return 0
		default:
			return v
		}
	})
}

func (st *SquadMenuState) getActionItems() []string {
	windowProps := st.windowMount.GetProps()
	if windowProps.Member.Name == "" {
		return []string{TextClose}
	}
	return []string{
		fmt.Sprintf("位置: %s", windowProps.Member.Position),
		fmt.Sprintf("戦闘: %s", windowProps.Member.Combat),
		fmt.Sprintf("回収: %s", windowProps.Member.ItemPickup),
		fmt.Sprintf("処理: %s", windowProps.Member.ItemHandling),
		"解雇",
		TextClose,
	}
}

func (st *SquadMenuState) handleMemberSelection(world w.World) {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "squad")
	if !ok {
		return
	}
	itemIndex := menuState.ItemIndex
	batchCount := len(props.BatchCommands)

	// 一括操作コマンドの処理
	if itemIndex < batchCount {
		st.executeBatchCommand(world, props.BatchCommands[itemIndex])
		return
	}

	// 隊員の個別選択
	memberIndex := itemIndex - batchCount
	if memberIndex >= len(props.Members) {
		return
	}

	st.subState = squadSubStateWindow
	st.windowMount = hooks.NewMount[squadWindowProps]()
	st.windowMount.SetProps(squadWindowProps{
		Member: props.Members[memberIndex],
	})
}

func (st *SquadMenuState) executeBatchCommand(world w.World, command string) {
	_, err := query.GetPlayerEntity(world)
	if err != nil {
		return
	}
	members := query.SquadMembers(world)

	switch command {
	case "集合":
		for _, m := range members {
			if ai := query.GetAI(world, m); ai != nil {
				ai.Movement = gc.MovementEscort
			}
		}
	case "全員待機":
		for _, m := range members {
			if ai := query.GetAI(world, m); ai != nil {
				ai.Movement = gc.MovementStationary
			}
		}
	}
}

func (st *SquadMenuState) executeWindowAction(world w.World) error {
	windowProps := st.windowMount.GetProps()
	actionIndex, ok := hooks.GetState[int](st.windowMount, "squad_window_index")
	if !ok {
		return nil
	}
	actionItems := st.getActionItems()
	if actionIndex >= len(actionItems) {
		return nil
	}

	member := windowProps.Member.Entity
	selectedAction := actionItems[actionIndex]

	ai := query.GetAI(world, member)
	if ai == nil {
		return nil
	}

	cycleAndRefresh := func(update func()) error {
		update()
		st.refreshWindowProps(world, member)
		return nil
	}

	switch {
	case strings.HasPrefix(selectedAction, "位置"):
		allPos := gc.AllSquadMovementPolicies()
		return cycleAndRefresh(func() {
			for i, v := range allPos {
				if v == ai.Movement {
					ai.Movement = allPos[(i+1)%len(allPos)]
					return
				}
			}
			ai.Movement = allPos[0]
		})

	case strings.HasPrefix(selectedAction, "戦闘"):
		allCombat := gc.AllSquadCombatPolicies()
		return cycleAndRefresh(func() {
			ai.CombatCurrent = allCombat[(int(ai.CombatCurrent)+1)%len(allCombat)]
		})

	case strings.HasPrefix(selectedAction, "回収"):
		all := gc.AllItemPickupPolicies()
		return cycleAndRefresh(func() {
			ai.ItemPickup = all[(int(ai.ItemPickup)+1)%len(all)]
		})

	case strings.HasPrefix(selectedAction, "処理"):
		all := gc.AllItemHandlingPolicies()
		return cycleAndRefresh(func() {
			ai.ItemHandling = all[(int(ai.ItemHandling)+1)%len(all)]
		})

	case selectedAction == "解雇":
		if err := lifecycle.DismissSquadMember(world, member); err != nil {
			return err
		}
		st.subState = squadSubStateMenu

	case selectedAction == TextClose:
		st.subState = squadSubStateMenu
	}

	return nil
}

func (st *SquadMenuState) refreshWindowProps(world w.World, member ecs.Entity) {
	name := query.GetEntityName(member, world)
	hp := world.Components.HP.Get(member).(*gc.HP)
	ai := query.GetAI(world, member)
	if ai == nil {
		return
	}

	st.windowMount.SetProps(squadWindowProps{
		Member: squadMemberData{
			Entity:       member,
			Name:         name,
			HP:           fmt.Sprintf("%d/%d", hp.Current, hp.Max),
			Position:     ai.Movement.String(),
			Combat:       ai.CombatCurrent.Label(),
			ItemPickup:   ai.ItemPickup.String(),
			ItemHandling: ai.ItemHandling.String(),
		},
	})
}

// ================
// buildUI
// ================

func (st *SquadMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "squad")
	itemIndex := menuState.ItemIndex

	root := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	root.AddChild(styled.NewTitleText("命令", res))
	root.AddChild(st.buildBatchCommands(props.BatchCommands, itemIndex, res))
	root.AddChild(st.buildMemberTable(props.Members, len(props.BatchCommands), itemIndex, res))

	eui := &ebitenui.UI{Container: root}

	if st.subState == squadSubStateWindow {
		window := st.buildActionWindow(world)
		eui.AddWindow(window)
	}

	return eui
}

func (st *SquadMenuState) buildBatchCommands(commands []string, selectedIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	for i, cmd := range commands {
		isSelected := i == selectedIndex
		container.AddChild(styled.NewListItemText(cmd, theme.TextSecondary, isSelected, res))
	}
	return container
}

func (st *SquadMenuState) buildMemberTable(members []squadMemberData, batchCount int, selectedIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()

	if len(members) == 0 {
		container.AddChild(styled.NewDescriptionText("隊員がいません", res))
		return container
	}

	columnWidths := []int{20, 120, 80, 60, 60}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight, styled.AlignLeft, styled.AlignLeft}

	// ヘッダー
	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(table, columnWidths, []string{"", "名前", "HP", "位置", "戦闘"}, res)

	for i, m := range members {
		isSelected := (i + batchCount) == selectedIndex
		styled.NewTableRow(table, columnWidths, []string{"", m.Name, m.HP, m.Position, m.Combat}, aligns, &isSelected, res)
	}

	container.AddChild(table)
	return container
}

func (st *SquadMenuState) buildActionWindow(world w.World) *widget.Window {
	res := world.Resources.UIResources
	actionIndex, _ := hooks.GetState[int](st.windowMount, "squad_window_index")
	actionItems := st.getActionItems()

	windowContainer := styled.NewWindowContainer(res)
	windowProps := st.windowMount.GetProps()
	titleContainer := styled.NewWindowHeaderContainer(windowProps.Member.Name, res)
	window := styled.NewSmallWindow(titleContainer, windowContainer)

	for i, action := range actionItems {
		isSelected := i == actionIndex
		windowContainer.AddChild(styled.NewListItemText(action, theme.TextSecondary, isSelected, res))
	}

	window.SetLocation(getCenterWinRect(world))
	return window
}
