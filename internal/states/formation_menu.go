package states

import (
	"fmt"

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
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// FormationMenuState は隊編成画面のゲームステート
type FormationMenuState struct {
	es.BaseState[w.World]
	menuMount *hooks.Mount[formationProps]
	widget    *ebitenui.UI
}

func (st FormationMenuState) String() string {
	return "FormationMenu"
}

var _ es.State[w.World] = &FormationMenuState{}
var _ es.ActionHandler[w.World] = &FormationMenuState{}

func (st *FormationMenuState) OnPause(_ w.World) error  { return nil }
func (st *FormationMenuState) OnResume(_ w.World) error { return nil }
func (st *FormationMenuState) OnStop(_ w.World) error   { return nil }
func (st *FormationMenuState) OnStart(_ w.World) error {
	st.menuMount = hooks.NewMount[formationProps]()
	return nil
}

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

func (st *FormationMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

func (st *FormationMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

func (st *FormationMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		st.toggleMemberActive(world)
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
	Status string // "同行" or "待機"
	Active bool
}

func (st *FormationMenuState) fetchProps(world w.World) formationProps {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return formationProps{}
	}

	var members []formationMemberData
	for _, member := range query.AllSquadMembers(world, playerEntity) {
		name := query.GetEntityName(member, world)
		hp := world.Components.HP.Get(member).(*gc.HP)
		sm := world.Components.SquadMember.Get(member).(*gc.SquadMember)

		status := "待機"
		if sm.Active {
			status = "同行"
		}

		members = append(members, formationMemberData{
			Entity: member,
			Name:   name,
			HP:     fmt.Sprintf("%d/%d", hp.Current, hp.Max),
			Status: status,
			Active: sm.Active,
		})
	}

	return formationProps{Members: members}
}

// ================
// Actions
// ================

func (st *FormationMenuState) toggleMemberActive(world w.World) {
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
	_ = lifecycle.SetSquadMemberActive(world, member.Entity, !member.Active)
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

	root.AddChild(styled.NewTitleText("編成", res))
	root.AddChild(st.buildMemberTable(props.Members, itemIndex, res))

	return &ebitenui.UI{Container: root}
}

func (st *FormationMenuState) buildMemberTable(members []formationMemberData, selectedIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()

	if len(members) == 0 {
		container.AddChild(styled.NewDescriptionText("隊員がいません", res))
		return container
	}

	columnWidths := []int{20, 120, 80, 60}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight, styled.AlignLeft}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(table, columnWidths, []string{"", "名前", "HP", "状態"}, res)

	for i, m := range members {
		isSelected := i == selectedIndex
		styled.NewTableRow(table, columnWidths, []string{"", m.Name, m.HP, m.Status}, aligns, &isSelected, res)
	}

	container.AddChild(table)
	return container
}
