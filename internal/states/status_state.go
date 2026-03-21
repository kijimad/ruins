package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

const statusItemsPerPage = 20

// StatusState はステータス画面のステート
type StatusState struct {
	es.BaseState[w.World]
	mount  *hooks.Mount[statusProps]
	widget *ebitenui.UI
}

func (st StatusState) String() string {
	return "Status"
}

var _ es.State[w.World] = &StatusState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *StatusState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *StatusState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *StatusState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *StatusState) OnStart(_ w.World) error {
	st.mount = hooks.NewMount[statusProps]()
	return nil
}

// Update はステートの更新処理
func (st *StatusState) Update(world w.World) (es.Transition[w.World], error) {
	action, ok := HandleMenuInput()
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
	hooks.UseTabMenu(st.mount.Store(), "status", hooks.TabMenuConfig{
		TabCount:     len(props.Tabs),
		ItemCounts:   itemCounts,
		ItemsPerPage: statusItemsPerPage,
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

// DoAction はActionを実行する
func (st *StatusState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev, inputmapper.ActionMenuSelect:
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
	BodyPart    gc.BodyPart
	Details     []statusDetailRow // 詳細パネルに表示する内訳
}

type statusDetailRow struct {
	Label string
	Value string
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

// タブID定数
const (
	tabBasic      = "basic"
	tabAttributes = "attributes"
	tabSkills     = "skills"
	tabEffects    = "effects"
	tabHealth     = "health"
)

func (st *StatusState) createTabs(world w.World, playerEntity ecs.Entity, envTemp int) []statusTabData {
	return []statusTabData{
		{ID: tabBasic, Label: "基本", Items: st.createBasicItems(world, playerEntity, envTemp)},
		{ID: tabAttributes, Label: "能力", Items: st.createAttributeItems(world, playerEntity)},
		{ID: tabSkills, Label: "スキル", Items: st.createSkillItems(world, playerEntity)},
		{ID: tabEffects, Label: "効果", Items: st.createEffectItems(world, playerEntity)},
		{ID: tabHealth, Label: "健康", Items: st.createHealthItems(world, playerEntity)},
	}
}

func (st *StatusState) createBasicItems(world w.World, playerEntity ecs.Entity, envTemp int) []statusItemData {
	items := []statusItemData{}

	if playerEntity.HasComponent(world.Components.Pools) {
		pools := world.Components.Pools.Get(playerEntity).(*gc.Pools)
		items = append(items,
			statusItemData{Label: "HP", Value: fmt.Sprintf("%d", pools.HP.Max), Description: "体力。0になると死亡する"},
			statusItemData{Label: "SP", Value: fmt.Sprintf("%d", pools.SP.Max), Description: "スタミナ。行動に消費する"},
			statusItemData{Label: "EP", Value: fmt.Sprintf("%d", pools.EP.Max), Description: "電力。電子機器の使用に消費する"},
			statusItemData{Label: "最大重量", Value: fmt.Sprintf("%.1fkg", pools.Weight.Max), Description: "所持可能な最大重量"},
		)
	}

	if playerEntity.HasComponent(world.Components.Hunger) {
		hunger := world.Components.Hunger.Get(playerEntity).(*gc.Hunger)
		items = append(items,
			statusItemData{Label: "空腹度", Value: fmt.Sprintf("%d (%s)", hunger.Current, hunger.GetLevel().String()), Description: "空腹度。高いと行動に支障が出る"},
		)
	}

	items = append(items,
		statusItemData{Label: "環境気温", Value: fmt.Sprintf("%d%s", envTemp, consts.IconDegree), Description: "現在地の気温"},
		statusItemData{Label: "時間帯", Value: world.Resources.Dungeon.GameTime.GetTimeOfDay().String(), Description: "現在の時間帯。屋外では気温に影響する"},
	)

	return items
}

func (st *StatusState) createAttributeItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}

	if playerEntity.HasComponent(world.Components.Attributes) {
		attrs := world.Components.Attributes.Get(playerEntity).(*gc.Attributes)
		items = append(items,
			statusItemData{Label: consts.VitalityLabel, Value: fmt.Sprintf("%d", attrs.Vitality.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Vitality.Modifier), Description: "体力。HPとSPの最大値に影響する"},
			statusItemData{Label: consts.StrengthLabel, Value: fmt.Sprintf("%d", attrs.Strength.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Strength.Modifier), Description: "筋力。近接攻撃のダメージに影響する"},
			statusItemData{Label: consts.SensationLabel, Value: fmt.Sprintf("%d", attrs.Sensation.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Sensation.Modifier), Description: "感覚。射撃攻撃のダメージに影響する"},
			statusItemData{Label: consts.DexterityLabel, Value: fmt.Sprintf("%d", attrs.Dexterity.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Dexterity.Modifier), Description: "器用さ。命中率に影響する"},
			statusItemData{Label: consts.AgilityLabel, Value: fmt.Sprintf("%d", attrs.Agility.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Agility.Modifier), Description: "敏捷。回避率と行動速度に影響する"},
			statusItemData{Label: consts.DefenseLabel, Value: fmt.Sprintf("%d", attrs.Defense.Total), Modifier: fmt.Sprintf("(%+d)", attrs.Defense.Modifier), Description: "防御。被ダメージを軽減する"},
		)
	}

	return items
}

func (st *StatusState) createSkillItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}

	if !playerEntity.HasComponent(world.Components.Skills) {
		return items
	}
	skills := world.Components.Skills.Get(playerEntity).(*gc.Skills)

	for _, id := range gc.AllSkillIDs {
		s, ok := skills.Data[id]
		if !ok {
			continue
		}
		name := gc.SkillName[id]
		expFrac := 0
		if s.Exp.Max > 0 {
			expFrac = s.Exp.Current * 1000 / s.Exp.Max
		}
		items = append(items, statusItemData{
			Label:       name,
			Value:       fmt.Sprintf("%d.%03d", s.Value, expFrac),
			Description: fmt.Sprintf("%s スキル", name),
		})
	}

	return items
}

func (st *StatusState) createEffectItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}

	if !playerEntity.HasComponent(world.Components.CharModifiers) {
		return items
	}
	e := world.Components.CharModifiers.Get(playerEntity).(*gc.CharModifiers)

	// 武器ダメージ倍率
	for _, id := range gc.AllSkillIDs {
		if mult, ok := e.WeaponDamage[id]; ok {
			items = append(items, statusItemData{
				Label:       gc.SkillName[id] + "攻撃力",
				Value:       fmt.Sprintf("%d%%", mult),
				Description: fmt.Sprintf("%s武器のダメージ倍率", gc.SkillName[id]),
				Details:     sourceToDetails(e.Sources, gc.WeaponDamageKeys[id]),
			})
		}
	}

	// 武器命中倍率
	for _, id := range gc.AllSkillIDs {
		if mult, ok := e.WeaponAccuracy[id]; ok {
			items = append(items, statusItemData{
				Label:       gc.SkillName[id] + "命中",
				Value:       fmt.Sprintf("%d%%", mult),
				Description: fmt.Sprintf("%s武器の命中倍率", gc.SkillName[id]),
				Details:     sourceToDetails(e.Sources, gc.WeaponAccuracyKeys[id]),
			})
		}
	}

	// 元素耐性倍率
	elementNames := map[gc.ElementType]string{
		gc.ElementTypeFire:    "火",
		gc.ElementTypeThunder: "雷",
		gc.ElementTypeChill:   "氷",
		gc.ElementTypePhoton:  "光",
	}
	elementKeys := map[gc.ElementType]gc.ModifierKey{
		gc.ElementTypeFire:    gc.ModFireResist,
		gc.ElementTypeThunder: gc.ModThunderResist,
		gc.ElementTypeChill:   gc.ModChillResist,
		gc.ElementTypePhoton:  gc.ModPhotonResist,
	}
	for _, elem := range []gc.ElementType{gc.ElementTypeFire, gc.ElementTypeThunder, gc.ElementTypeChill, gc.ElementTypePhoton} {
		if mult, ok := e.ElementResist[elem]; ok {
			items = append(items, statusItemData{
				Label:       elementNames[elem] + "耐性",
				Value:       fmt.Sprintf("%d%%", mult),
				Description: fmt.Sprintf("%s属性ダメージの倍率。低いほど軽減される", elementNames[elem]),
				Details:     sourceToDetails(e.Sources, elementKeys[elem]),
			})
		}
	}

	// その他の効果倍率
	items = append(items,
		statusItemData{Label: "低体温進行", Value: fmt.Sprintf("%d%%", e.ColdProgress), Description: "低体温の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModColdProgress)},
		statusItemData{Label: "高体温進行", Value: fmt.Sprintf("%d%%", e.HeatProgress), Description: "高体温の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModHeatProgress)},
		statusItemData{Label: "空腹進行", Value: fmt.Sprintf("%d%%", e.HungerProgress), Description: "空腹の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModHungerProgress)},
		statusItemData{Label: "回復効果", Value: fmt.Sprintf("%d%%", e.HealingEffect), Description: "回復アイテムの効果倍率。高いほど多く回復する", Details: sourceToDetails(e.Sources, gc.ModHealingEffect)},
		statusItemData{Label: "最大重量", Value: fmt.Sprintf("%d%%", e.MaxWeight), Description: "所持可能な最大重量の倍率", Details: sourceToDetails(e.Sources, gc.ModMaxWeight)},
		statusItemData{Label: "発見", Value: fmt.Sprintf("%d%%", e.Exploration), Description: "アイテム発見率の倍率。高いほど見つけやすい", Details: sourceToDetails(e.Sources, gc.ModExploration)},
		statusItemData{Label: "被発見", Value: fmt.Sprintf("%d%%", e.EnemyVision), Description: "敵に発見される距離の倍率。低いほど見つかりにくい", Details: sourceToDetails(e.Sources, gc.ModEnemyVision)},
		statusItemData{Label: "暗所視界", Value: fmt.Sprintf("%d%%", e.NightVision), Description: "暗所での視界の倍率。高いほど見える", Details: sourceToDetails(e.Sources, gc.ModNightVision)},
		statusItemData{Label: "移動速度", Value: fmt.Sprintf("%d%%", e.MoveCost), Description: "移動時のAPコスト倍率。低いほど少ないAPで移動できる", Details: sourceToDetails(e.Sources, gc.ModMoveCost)},
		statusItemData{Label: "素材消費", Value: fmt.Sprintf("%d%%", e.CraftCost), Description: "合成時の素材消費量倍率。低いほど素材が節約できる", Details: sourceToDetails(e.Sources, gc.ModCraftCost)},
		statusItemData{Label: "合成品質", Value: fmt.Sprintf("%d%%", e.SmithQuality), Description: "調合時の品質倍率。高いほど良い品ができる", Details: sourceToDetails(e.Sources, gc.ModSmithQuality)},
		statusItemData{Label: "買値", Value: fmt.Sprintf("%d%%", e.BuyPrice), Description: "買い物の価格倍率。低いほど安く買える", Details: sourceToDetails(e.Sources, gc.ModBuyPrice)},
		statusItemData{Label: "売値", Value: fmt.Sprintf("%d%%", e.SellPrice), Description: "売却の価格倍率。高いほど高く売れる", Details: sourceToDetails(e.Sources, gc.ModSellPrice)},
		statusItemData{Label: "最大荷重", Value: fmt.Sprintf("%d%%", e.HeavyArmor), Description: "最大荷重倍率", Details: sourceToDetails(e.Sources, gc.ModHeavyArmor)},
	)

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
			BodyPart:    part,
		})
	}

	return items
}

func (st *StatusState) getHealthPartDescription(part gc.BodyPart) string {
	switch part {
	case gc.BodyPartTorso:
		return "胴体"
	case gc.BodyPartHead:
		return "頭部"
	case gc.BodyPartArms:
		return "腕"
	case gc.BodyPartHands:
		return "手"
	case gc.BodyPartLegs:
		return "脚"
	case gc.BodyPartFeet:
		return "足"
	case gc.BodyPartWholeBody:
		return "全身"
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
	tabIndex, _ := hooks.GetState[int](st.mount, "status_tabIndex")
	itemIndex, _ := hooks.GetState[int](st.mount, "status_itemIndex")

	root := styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	root.AddChild(styled.NewTitleText("ステータス", res))
	root.AddChild(st.buildCategoryContainer(props.Tabs, tabIndex, res))
	root.AddChild(widget.NewContainer())

	// 一覧と詳細を横並びにする
	midRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	midRow.AddChild(st.buildItemContainer(props.Tabs, tabIndex, itemIndex, res))
	midRow.AddChild(st.buildDetailContainer(world, props, tabIndex, itemIndex, res))
	root.AddChild(midRow)
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

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
	pg := pagination.New(itemIndex, len(currentTab.Items), statusItemsPerPage)

	// ページインジケーター
	pageText := pg.GetPageText()
	if pageText == "" {
		pageText = " "
	}
	container.AddChild(styled.NewPageIndicator(pageText, res))

	columnWidths := []int{20, 100, 60, 60}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight, styled.AlignRight}

	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
		isSelected := pg.IsSelectedInPage(entry.Index)
		styled.NewTableRow(table, columnWidths, []string{"", entry.Item.Label, entry.Item.Value, entry.Item.Modifier}, aligns, &isSelected, res)
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

	if tabIndex >= len(props.Tabs) {
		return container
	}
	if itemIndex >= len(props.Tabs[tabIndex].Items) {
		return container
	}

	tabID := props.Tabs[tabIndex].ID
	item := props.Tabs[tabIndex].Items[itemIndex]

	switch tabID {
	case tabEffects:
		st.buildEffectDetail(container, item, res)
	case tabHealth:
		st.buildHealthDetail(container, item, world, res)
	}

	return container
}

func (st *StatusState) buildEffectDetail(container *widget.Container, item statusItemData, res *resources.UIResources) {
	if len(item.Details) == 0 {
		return
	}

	columnWidths := []int{100, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(table, columnWidths, []string{"内訳", ""}, res)
	for _, d := range item.Details {
		styled.NewTableRow(table, columnWidths, []string{d.Label, d.Value}, aligns, nil, res)
	}
	container.AddChild(table)
}

func (st *StatusState) buildHealthDetail(container *widget.Container, item statusItemData, world w.World, res *resources.UIResources) {
	var playerEntity ecs.Entity
	worldhelper.QueryPlayer(world, func(entity ecs.Entity) {
		playerEntity = entity
	})
	insulation := systems.CalculateEquippedInsulation(world, playerEntity)
	lowerBound, upperBound := systems.ComfortableRange(insulation)

	columnWidths := []int{80, 100}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}

	tempTable := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(tempTable, columnWidths, []string{"快適温度", ""}, res)
	styled.NewTableRow(tempTable, columnWidths, []string{"範囲", fmt.Sprintf("%d%s 〜 %d%s", lowerBound, consts.IconDegree, upperBound, consts.IconDegree)}, aligns, nil, res)
	container.AddChild(tempTable)
}

// sourceToDetails はModifierSourceのスライスから内訳表示用の行を生成する。
// 変化量が0のソースは表示しない。
func sourceToDetails(sources map[gc.ModifierKey][]gc.ModifierSource, key gc.ModifierKey) []statusDetailRow {
	srcs, ok := sources[key]
	if !ok {
		return nil
	}
	var rows []statusDetailRow
	for _, s := range srcs {
		if s.Value == 0 {
			continue
		}
		rows = append(rows, statusDetailRow{
			Label: s.Label,
			Value: fmt.Sprintf("%+d%%", s.Value),
		})
	}
	return rows
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
