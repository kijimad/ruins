package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// MemberStatusState は隊員のステータス詳細画面
type MemberStatusState struct {
	es.BaseState[w.World]
	member ecs.Entity
	mount  *hooks.Mount[memberStatusProps]
	widget *ebitenui.UI
}

func (st MemberStatusState) String() string {
	return "MemberStatus"
}

var _ es.State[w.World] = &MemberStatusState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *MemberStatusState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *MemberStatusState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *MemberStatusState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始する際に呼ばれる
func (st *MemberStatusState) OnStart(_ w.World) error {
	st.mount = hooks.NewMount[memberStatusProps]()
	return nil
}

// Update はステートの更新処理を行う
func (st *MemberStatusState) Update(world w.World) (es.Transition[w.World], error) {
	action, ok := HandleMenuInput()
	if ok {
		switch action {
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			return es.Transition[w.World]{Type: es.TransPop}, nil
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuSelect, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
			st.mount.Dispatch(action)
		default:
			return es.Transition[w.World]{}, fmt.Errorf("memberStatus: 未対応のアクション: %s", action)
		}
	}

	props := st.fetchProps(world)
	st.mount.SetProps(props)

	hooks.UseTabMenu(st.mount.Store(), "member_status", hooks.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: props.itemCounts(),
	})

	if st.mount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理を行う
func (st *MemberStatusState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// ================
// Props
// ================

type memberStatusProps struct {
	Name string
	Tabs []memberStatusTab
}

type memberStatusTab struct {
	Label string
	Items []memberStatusItem
}

type memberStatusItem struct {
	Label string
	Value string
}

func (p memberStatusProps) itemCounts() []int {
	counts := make([]int, len(p.Tabs))
	for i, tab := range p.Tabs {
		counts[i] = len(tab.Items)
	}
	return counts
}

func (st *MemberStatusState) fetchProps(world w.World) memberStatusProps {
	member := st.member
	name := query.GetEntityName(member, world)

	return memberStatusProps{
		Name: name,
		Tabs: []memberStatusTab{
			st.fetchStatusTab(world, member),
		},
	}
}

func (st *MemberStatusState) fetchStatusTab(world w.World, member ecs.Entity) memberStatusTab {
	var items []memberStatusItem

	if world.Components.HP.Has(member) {
		hp := world.Components.HP.Get(member)
		items = append(items, memberStatusItem{Label: "HP", Value: fmt.Sprintf("%d / %d", hp.Current, hp.Max)})
	}

	if world.Components.Abilities.Has(member) {
		abils := world.Components.Abilities.Get(member)
		items = append(items,
			memberStatusItem{Label: consts.VitalityLabel, Value: fmt.Sprintf("%d", abils.Vitality.Base)},
			memberStatusItem{Label: consts.StrengthLabel, Value: fmt.Sprintf("%d", abils.Strength.Base)},
			memberStatusItem{Label: consts.SensationLabel, Value: fmt.Sprintf("%d", abils.Sensation.Base)},
			memberStatusItem{Label: consts.DexterityLabel, Value: fmt.Sprintf("%d", abils.Dexterity.Base)},
			memberStatusItem{Label: consts.AgilityLabel, Value: fmt.Sprintf("%d", abils.Agility.Base)},
			memberStatusItem{Label: consts.DefenseLabel, Value: fmt.Sprintf("%d", abils.Defense.Base)},
		)
	}

	if world.Components.WeightCapacity.Has(member) {
		wc := world.Components.WeightCapacity.Get(member)
		items = append(items, memberStatusItem{Label: "重量", Value: fmt.Sprintf("%.1f / %.1f", wc.Current, wc.Max)})
	}

	return memberStatusTab{
		Label: "ステータス",
		Items: items,
	}
}

// ================
// buildUI
// ================

func (st *MemberStatusState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, "member_status")
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	root := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	root.AddChild(styled.NewTitleText(props.Name, res))
	root.AddChild(st.buildTabBar(props.Tabs, tabIndex, res))

	if tabIndex < len(props.Tabs) {
		root.AddChild(st.buildItemTable(props.Tabs[tabIndex].Items, itemIndex, res))
	}

	return &ebitenui.UI{Container: root}
}

func (st *MemberStatusState) buildTabBar(tabs []memberStatusTab, tabIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	for i, tab := range tabs {
		isSelected := i == tabIndex
		color := theme.TextSecondary
		if isSelected {
			color = theme.TextPrimary
		}
		container.AddChild(styled.NewListItemText(tab.Label, color, isSelected, res))
	}
	return container
}

func (st *MemberStatusState) buildItemTable(items []memberStatusItem, selectedIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()

	columnWidths := []int{20, 100, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight}

	table := styled.NewTableContainer(columnWidths, res)
	for i, item := range items {
		isSelected := i == selectedIndex
		styled.NewTableRow(table, columnWidths, []string{"", item.Label, item.Value}, aligns, &isSelected, res)
	}

	container.AddChild(table)
	return container
}
