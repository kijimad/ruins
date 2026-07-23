package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// FormationMenuState は隊編成画面のゲームステート
type FormationMenuState struct {
	es.BaseState[w.World]
	menuMount *hooks.Mount[formationProps]
	widget    *ebitenui.UI
}

var _ es.State[w.World] = &FormationMenuState{}
var _ es.ActionHandler[w.World] = &FormationMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *FormationMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *FormationMenuState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *FormationMenuState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始する際に呼ばれる
func (st *FormationMenuState) OnStart(_ w.World) error {
	st.menuMount = hooks.NewMount[formationProps]()
	return nil
}

// Update はステートの更新処理を行う
func (st *FormationMenuState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.menuMount.Dispatch(action)
	}

	props := st.fetchProps(world)
	st.menuMount.SetProps(props)

	hooks.UseTabMenu(st.menuMount.Store(), "formation", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Members)},
	})

	menuDirty := st.menuMount.Update()
	if menuDirty || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理を行う
func (st *FormationMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// HandleInput は入力を処理してアクションIDを返す
func (st *FormationMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はアクションを実行してステート遷移を返す
func (st *FormationMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		st.showMemberDetail()
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理
	default:
		return es.Transition[w.World]{}, fmt.Errorf("formationMenu: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

type formationProps struct {
	Members []formationMemberData
}

type formationMemberData struct {
	Entity ecs.Entity
	Name   string
	HP     string
}

func (st *FormationMenuState) fetchProps(world w.World) formationProps {
	_, err := query.GetPlayerEntity(world)
	if err != nil {
		return formationProps{}
	}

	var members []formationMemberData
	for _, member := range query.SquadMembers(world) {
		name := query.GetEntityName(member, world)
		hp := world.Components.HP.Get(member)

		members = append(members, formationMemberData{
			Entity: member,
			Name:   name,
			HP:     fmt.Sprintf("%d/%d", hp.Current, hp.Max),
		})
	}

	return formationProps{Members: members}
}

// ================
// Actions
// ================

func (st *FormationMenuState) showMemberDetail() {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "formation")
	if !ok {
		return
	}
	itemIndex := menuState.ItemIndex
	if itemIndex >= len(props.Members) {
		return
	}

	member := props.Members[itemIndex]
	st.SetTransition(es.Transition[w.World]{
		Type:          es.TransPush,
		NewStateFuncs: []es.StateFactory[w.World]{func() (es.State[w.World], error) { return NewMemberStatusState(member.Entity) }},
	})
}

// ================
// buildUI
// ================

func (st *FormationMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "formation")
	itemIndex := menuState.ItemIndex

	root := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	root.AddChild(styled.NewTitleText("部隊", res))
	root.AddChild(st.buildMemberTable(props.Members, itemIndex, res))

	return &ebitenui.UI{Container: root}
}

func (st *FormationMenuState) buildMemberTable(members []formationMemberData, selectedIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()

	if len(members) == 0 {
		container.AddChild(styled.NewDescriptionText("隊員がいません", res))
		return container
	}

	columnWidths := []int{20, 120, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(table, columnWidths, []string{"", "名前", "HP"}, res)

	for i, m := range members {
		isSelected := i == selectedIndex
		styled.NewTableRow(table, columnWidths, []string{"", m.Name, m.HP}, aligns, &isSelected, res)
	}

	container.AddChild(table)
	return container
}
