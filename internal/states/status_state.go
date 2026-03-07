package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// StatusState はステータス画面のステート
type StatusState struct {
	es.BaseState[w.World]
	mount  *ui.Mount[statusProps]
	widget *ebitenui.UI
}

func (st StatusState) String() string {
	return "Status"
}

var _ es.State[w.World] = &StatusState{}
var _ es.ActionHandler[w.World] = &StatusState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *StatusState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *StatusState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *StatusState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *StatusState) OnStart(_ w.World) error {
	st.mount = ui.NewMount[statusProps]()
	return nil
}

// Update はステートの更新処理
func (st *StatusState) Update(world w.World) (es.Transition[w.World], error) {
	action, ok := st.HandleInput(world.Config)
	if ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.mount.Dispatch(action)
	}

	props := st.fetchProps(world)
	st.mount.SetProps(props)

	// UseTabMenuを呼んでreducerを登録・更新
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	ui.UseTabMenu(st.mount.Store(), "status", ui.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})

	if st.mount.Update() {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理
func (st *StatusState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *StatusState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *StatusState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// NewStatusState はステータス画面のStateを作成する
func NewStatusState() es.State[w.World] {
	return &StatusState{}
}

// ================
// Props
// ================

type statusProps struct {
	Tabs []statusTabData
}

type statusTabData struct {
	ID    string
	Label string
	Items []statusItemData
}

type statusItemData struct {
	Label       string
	Value       string
	Modifier    string
	Description string
	TabID       string
	BodyPart    gc.BodyPart
}

func (st *StatusState) fetchProps(world w.World) statusProps {
	var playerEntity ecs.Entity
	worldhelper.QueryPlayer(world, func(entity ecs.Entity) {
		playerEntity = entity
	})

	envTemp := 0
	if playerEntity.HasComponent(world.Components.GridElement) {
		gridElement := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)
		temp, err := systems.CalculateEnvTemperature(world, gridElement.X, gridElement.Y)
		if err == nil {
			envTemp = temp
		}
	}

	return statusProps{
		Tabs: st.createTabs(world, playerEntity, envTemp),
	}
}

func (st *StatusState) createTabs(world w.World, playerEntity ecs.Entity, envTemp int) []statusTabData {
	return []statusTabData{
		{ID: "basic", Label: "基本", Items: st.createBasicItems(world, playerEntity, envTemp)},
		{ID: "attributes", Label: "能力", Items: st.createAttributeItems(world, playerEntity)},
		{ID: "health", Label: "健康", Items: st.createHealthItems(world, playerEntity)},
	}
}

func (st *StatusState) createBasicItems(world w.World, playerEntity ecs.Entity, envTemp int) []statusItemData {
	items := []statusItemData{}

	if playerEntity.HasComponent(world.Components.Pools) {
		pools := world.Components.Pools.Get(playerEntity).(*gc.Pools)
		items = append(items,
			statusItemData{Label: "HP", Value: fmt.Sprintf("%d", pools.HP.Max), Description: "体力。0になると死亡する", TabID: "basic"},
			statusItemData{Label: "SP", Value: fmt.Sprintf("%d", pools.SP.Max), Description: "スタミナ。行動に消費する", TabID: "basic"},
			statusItemData{Label: "EP", Value: fmt.Sprintf("%d", pools.EP.Max), Description: "電力。電子機器の使用に消費する", TabID: "basic"},
			statusItemData{Label: "最大重量", Value: fmt.Sprintf("%.1fkg", pools.Weight.Max), Description: "所持可能な最大重量", TabID: "basic"},
		)
	}

	if playerEntity.HasComponent(world.Components.Hunger) {
		hunger := world.Components.Hunger.Get(playerEntity).(*gc.Hunger)
		items = append(items,
			statusItemData{Label: "空腹度", Value: fmt.Sprintf("%d (%s)", hunger.Current, hunger.GetLevel().String()), Description: "空腹度。高いと行動に支障が出る", TabID: "basic"},
		)
	}

	items = append(items,
		statusItemData{Label: "環境気温", Value: fmt.Sprintf("%d%s", envTemp, consts.IconDegree), Description: "現在地の気温", TabID: "basic"},
		statusItemData{Label: "時間帯", Value: world.Resources.Dungeon.GameTime.GetTimeOfDay().String(), Description: "現在の時間帯。屋外では気温に影響する", TabID: "basic"},
	)

	return items
}

func (st *StatusState) createAttributeItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}

	if playerEntity.HasComponent(world.Components.Attributes) {
		attrs := world.Components.Attributes.Get(playerEntity).(*gc.Attributes)
		items = append(items,
			statusItemData{Label: consts.VitalityLabel, Value: fmt.Sprintf("%d", attrs.Vitality.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Vitality.Modifier), Description: "体力。HPとSPの最大値に影響する", TabID: "attributes"},
			statusItemData{Label: consts.StrengthLabel, Value: fmt.Sprintf("%d", attrs.Strength.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Strength.Modifier), Description: "筋力。近接攻撃のダメージに影響する", TabID: "attributes"},
			statusItemData{Label: consts.SensationLabel, Value: fmt.Sprintf("%d", attrs.Sensation.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Sensation.Modifier), Description: "感覚。射撃攻撃のダメージに影響する", TabID: "attributes"},
			statusItemData{Label: consts.DexterityLabel, Value: fmt.Sprintf("%d", attrs.Dexterity.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Dexterity.Modifier), Description: "器用さ。命中率に影響する", TabID: "attributes"},
			statusItemData{Label: consts.AgilityLabel, Value: fmt.Sprintf("%d", attrs.Agility.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Agility.Modifier), Description: "敏捷。回避率と行動速度に影響する", TabID: "attributes"},
			statusItemData{Label: consts.DefenseLabel, Value: fmt.Sprintf("%d", attrs.Defense.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Defense.Modifier), Description: "防御。被ダメージを軽減する", TabID: "attributes"},
		)
	}

	return items
}

func (st *StatusState) createHealthItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}

	var hs *gc.HealthStatus
	if playerEntity.HasComponent(world.Components.HealthStatus) {
		hs = world.Components.HealthStatus.Get(playerEntity).(*gc.HealthStatus)
	}

	for i := 0; i < int(gc.BodyPartCount); i++ {
		part := gc.BodyPart(i)

		conditionStr := ""
		if hs != nil {
			conditions := hs.Parts[i].Conditions
			for j, cond := range conditions {
				if j > 0 {
					conditionStr += ", "
				}
				conditionStr += cond.DisplayName()
			}
		}

		value := conditionStr
		if value == "" {
			value = "正常"
		}

		items = append(items, statusItemData{
			Label:       part.String(),
			Value:       value,
			Description: st.getHealthPartDescription(part),
			TabID:       "health",
			BodyPart:    part,
		})
	}

	return items
}

func (st *StatusState) getHealthPartDescription(part gc.BodyPart) string {
	switch part {
	case gc.BodyPartTorso:
		return "胴体。低体温で筋力と体力が低下する"
	case gc.BodyPartHead:
		return "頭部。低体温で感覚が低下する"
	case gc.BodyPartArms:
		return "腕。低体温で筋力が低下する"
	case gc.BodyPartHands:
		return "手。低体温で器用さが低下し、凍傷のリスクがある"
	case gc.BodyPartLegs:
		return "脚。低体温で敏捷が低下する"
	case gc.BodyPartFeet:
		return "足。低体温で敏捷が低下し、凍傷のリスクがある"
	default:
		return ""
	}
}

// ================
// buildUI
// ================

func (st *StatusState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()
	tabIndex, _ := ui.GetState[int](st.mount, "status_tabIndex")
	itemIndex, _ := ui.GetState[int](st.mount, "status_itemIndex")

	root := styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	root.AddChild(styled.NewTitleText("ステータス", res))
	root.AddChild(st.buildCategoryContainer(props.Tabs, tabIndex, res))
	root.AddChild(widget.NewContainer())

	root.AddChild(st.buildItemContainer(props.Tabs, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(st.buildDetailContainer(world, props, tabIndex, itemIndex, res))

	root.AddChild(st.buildDescContainer(props.Tabs, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	return &ebitenui.UI{Container: root}
}

func (st *StatusState) buildCategoryContainer(tabs []statusTabData, tabIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	for i, tab := range tabs {
		isSelected := i == tabIndex
		color := consts.ForegroundColor
		if isSelected {
			color = consts.TextColor
		}
		container.AddChild(styled.NewListItemText(tab.Label, color, isSelected, res))
	}
	return container
}

func (st *StatusState) buildItemContainer(tabs []statusTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(tabs) {
		return container
	}

	currentTab := tabs[tabIndex]
	columnWidths := []int{20, 100, 60, 60}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight, styled.AlignRight}

	table := styled.NewTableContainer(columnWidths, res)
	for i, item := range currentTab.Items {
		isSelected := i == itemIndex
		styled.NewTableRow(table, columnWidths, []string{"", item.Label, item.Value, item.Modifier}, aligns, &isSelected, res)
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(項目なし)", res))
	}

	return container
}

func (st *StatusState) buildDetailContainer(world w.World, props statusProps, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	if tabIndex >= len(props.Tabs) || props.Tabs[tabIndex].ID != "health" {
		return container
	}
	if itemIndex >= len(props.Tabs[tabIndex].Items) {
		return container
	}

	item := props.Tabs[tabIndex].Items[itemIndex]

	var playerEntity ecs.Entity
	worldhelper.QueryPlayer(world, func(entity ecs.Entity) {
		playerEntity = entity
	})
	allInsulation := systems.CalculateEquippedInsulation(world, playerEntity)
	lowerBound, upperBound := systems.ComfortableRange(allInsulation[item.BodyPart])

	columnWidths := []int{80, 100}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}

	tempTable := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(tempTable, columnWidths, []string{"快適温度", ""}, res)
	styled.NewTableRow(tempTable, columnWidths, []string{"範囲", fmt.Sprintf("%d%s 〜 %d%s", lowerBound, consts.IconDegree, upperBound, consts.IconDegree)}, aligns, nil, res)
	container.AddChild(tempTable)

	return container
}

func (st *StatusState) buildDescContainer(tabs []statusTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	desc := " "
	if tabIndex < len(tabs) && itemIndex < len(tabs[tabIndex].Items) {
		desc = tabs[tabIndex].Items[itemIndex].Description
	}
	if desc == "" {
		desc = " "
	}
	container.AddChild(styled.NewMenuText(desc, res))
	return container
}
