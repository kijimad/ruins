package states

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
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
	skips := make([][]bool, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
		s := make([]bool, len(tab.Items))
		for j, item := range tab.Items {
			s[j] = item.IsHeader
		}
		skips[i] = s
	}
	hooks.UseTabMenu(st.mount.Store(), "status", hooks.TabMenuConfig{
		TabCount:     len(props.Tabs),
		ItemCounts:   itemCounts,
		ItemsPerPage: statusItemsPerPage,
		Skips:        skips,
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
func NewStatusState() (es.State[w.World], error) {
	return &StatusState{}, nil
}

// ================
// Props
// ================

type statusProps struct {
	PlayerName string
	Tabs       []statusTabData
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
	IsHeader    bool // カテゴリヘッダー行かどうか
	BodyPart    gc.BodyPart
	Details     []statusDetailRow // 詳細パネルに表示する内訳
}

type statusDetailRow struct {
	Label string
	Value string
}

// aliveHas はエンティティが生存しコンポーネントを保持する場合のみtrueを返す。
// Arkは死亡エンティティのHasでパニックするため、生存確認と組み合わせる
func aliveHas[T any](world w.World, comp *ecs.Map[T], entity ecs.Entity) bool {
	return world.ECS.Alive(entity) && comp.Has(entity)
}

func (st *StatusState) fetchProps(world w.World) statusProps {
	var playerEntity ecs.Entity
	query.Player(world, func(entity ecs.Entity) {
		playerEntity = entity
	})

	envTemp := 0
	if aliveHas(world, world.Components.GridElement, playerEntity) {
		gridElement := world.Components.GridElement.Get(playerEntity)
		temp, err := systems.CalculateEnvTemperature(world, gridElement.X, gridElement.Y)
		if err == nil {
			envTemp = temp
		}
	}

	playerName := ""
	if aliveHas(world, world.Components.Name, playerEntity) {
		playerName = world.Components.Name.Get(playerEntity).Name
	}
	professionName := ""
	if aliveHas(world, world.Components.Profession, playerEntity) {
		profComp := world.Components.Profession.Get(playerEntity)
		if prof, err := raw.GetProfession(world.Resources.RawMaster, profComp.ID); err == nil {
			professionName = prof.Name
		}
	}

	return statusProps{
		PlayerName: playerName,
		Tabs:       st.createTabs(world, playerEntity, envTemp, professionName),
	}
}

// タブID定数
const (
	tabBasic     = "basic"
	tabAbilities = "abilities"
	tabSkills    = "skills"
	tabEffects   = "effects"
	tabHealth    = "health"
)

func (st *StatusState) createTabs(world w.World, playerEntity ecs.Entity, envTemp int, professionName string) []statusTabData {
	return []statusTabData{
		{ID: tabBasic, Label: "基本", Items: st.createBasicItems(world, playerEntity, envTemp, professionName)},
		{ID: tabAbilities, Label: "能力", Items: st.createAbilityItems(world, playerEntity)},
		{ID: tabSkills, Label: "スキル", Items: st.createSkillItems(world, playerEntity)},
		{ID: tabEffects, Label: "効果", Items: st.createEffectItems(world, playerEntity)},
		{ID: tabHealth, Label: "健康", Items: st.createHealthItems(world, playerEntity)},
	}
}

func (st *StatusState) createBasicItems(world w.World, playerEntity ecs.Entity, envTemp int, professionName string) []statusItemData {
	items := []statusItemData{}

	if professionName != "" {
		items = append(items, statusItemData{Label: "職業", Value: professionName, Description: "職業"})
	}

	if aliveHas(world, world.Components.HP, playerEntity) {
		hp := world.Components.HP.Get(playerEntity)
		items = append(items,
			statusItemData{Label: "HP", Value: fmt.Sprintf("%d", hp.Max), Description: "体力。0になると死亡する"},
		)
	}
	if aliveHas(world, world.Components.WeightCapacity, playerEntity) {
		cw := world.Components.WeightCapacity.Get(playerEntity)
		items = append(items,
			statusItemData{Label: "最大重量", Value: fmt.Sprintf("%.1f%s", cw.Max, consts.IconKg), Description: "所持可能な最大重量"},
		)
	}

	if aliveHas(world, world.Components.Hunger, playerEntity) {
		hunger := world.Components.Hunger.Get(playerEntity)
		items = append(items,
			statusItemData{Label: "空腹度", Value: hunger.GetLevel().String(), Description: "空腹度。高いと行動に支障が出る"},
		)
	}

	items = append(items,
		statusItemData{Label: "環境気温", Value: fmt.Sprintf("%d%s", envTemp, consts.IconDegree), Description: "現在地の気温"},
		statusItemData{Label: "時間帯", Value: query.GetDungeon(world).GameTime.GetTimeOfDay().String(), Description: "現在の時間帯。屋外では気温に影響する"},
	)

	return items
}

func (st *StatusState) createAbilityItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}

	if aliveHas(world, world.Components.Abilities, playerEntity) {
		abils := world.Components.Abilities.Get(playerEntity)
		items = append(items,
			statusItemData{Label: consts.VitalityLabel, Value: fmt.Sprintf("%d", abils.Vitality.Total), Modifier: fmt.Sprintf("(%+d)", abils.Vitality.Modifier), Description: "体力。HPとSPの最大値に影響する"},
			statusItemData{Label: consts.StrengthLabel, Value: fmt.Sprintf("%d", abils.Strength.Total), Modifier: fmt.Sprintf("(%+d)", abils.Strength.Modifier), Description: "筋力。近接攻撃のダメージに影響する"},
			statusItemData{Label: consts.SensationLabel, Value: fmt.Sprintf("%d", abils.Sensation.Total), Modifier: fmt.Sprintf("(%+d)", abils.Sensation.Modifier), Description: "感覚。射撃攻撃のダメージに影響する"},
			statusItemData{Label: consts.DexterityLabel, Value: fmt.Sprintf("%d", abils.Dexterity.Total), Modifier: fmt.Sprintf("(%+d)", abils.Dexterity.Modifier), Description: "器用さ。命中率に影響する"},
			statusItemData{Label: consts.AgilityLabel, Value: fmt.Sprintf("%d", abils.Agility.Total), Modifier: fmt.Sprintf("(%+d)", abils.Agility.Modifier), Description: "敏捷。回避率と行動速度に影響する"},
			statusItemData{Label: consts.DefenseLabel, Value: fmt.Sprintf("%d", abils.Defense.Total), Modifier: fmt.Sprintf("(%+d)", abils.Defense.Modifier), Description: "防御。被ダメージを軽減する"},
		)
	}

	return items
}

func (st *StatusState) createSkillItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}

	if !aliveHas(world, world.Components.Skills, playerEntity) {
		return items
	}
	skills := world.Components.Skills.Get(playerEntity)

	for _, cat := range gc.SkillCategories {
		// カテゴリヘッダーをアイテムとして挿入する
		items = append(items, statusItemData{
			Label:       cat.Name,
			IsHeader:    true,
			Description: fmt.Sprintf("%sカテゴリのスキル", cat.Name),
		})
		for _, id := range cat.IDs {
			s := skills.Get(id)
			name := gc.SkillName(id)
			expFrac := 0
			if s.Exp.Max > 0 {
				expFrac = s.Exp.Current * 1000 / s.Exp.Max
			}
			info := gc.SkillDescription(id)
			items = append(items, statusItemData{
				Label:       name,
				Value:       fmt.Sprintf("%d.%03d", s.Value, expFrac),
				Description: info.Summary,
				Details: []statusDetailRow{
					{Label: "獲得条件", Value: info.GainedBy},
					{Label: "効果", Value: info.Effect},
				},
			})
		}
	}

	return items
}

func (st *StatusState) createEffectItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}

	if !aliveHas(world, world.Components.CharModifiers, playerEntity) {
		return items
	}
	e := world.Components.CharModifiers.Get(playerEntity)

	// 戦闘
	items = append(items, statusItemData{Label: "戦闘", IsHeader: true, Description: "戦闘に関する効果"})
	for _, id := range gc.AllSkillIDs {
		if mult, ok := e.WeaponDamage[id]; ok {
			name := gc.SkillName(id)
			items = append(items, statusItemData{
				Label:       name + "攻撃力",
				Value:       fmt.Sprintf("%d%%", mult),
				Description: fmt.Sprintf("%s武器のダメージ倍率", name),
				Details:     sourceToDetails(e.Sources, gc.WeaponDamageKey(id)),
			})
		}
	}
	for _, id := range gc.AllSkillIDs {
		if mult, ok := e.WeaponAccuracy[id]; ok {
			name := gc.SkillName(id)
			items = append(items, statusItemData{
				Label:       name + "命中",
				Value:       fmt.Sprintf("%d%%", mult),
				Description: fmt.Sprintf("%s武器の命中倍率", name),
				Details:     sourceToDetails(e.Sources, gc.WeaponAccuracyKey(id)),
			})
		}
	}
	for _, elem := range []gc.ElementType{gc.ElementTypeFire, gc.ElementTypeThunder, gc.ElementTypeChill, gc.ElementTypePhoton} {
		if mult, ok := e.ElementResist[elem]; ok {
			items = append(items, statusItemData{
				Label:       elem.String() + "耐性",
				Value:       fmt.Sprintf("%d%%", mult),
				Description: fmt.Sprintf("%s属性ダメージの倍率。低いほど軽減される", elem.String()),
				Details:     sourceToDetails(e.Sources, gc.ElementResistKey(elem)),
			})
		}
	}

	// 生存
	items = append(items, statusItemData{Label: "生存", IsHeader: true, Description: "生存に関する効果"})
	items = append(items,
		statusItemData{Label: "低体温進行", Value: fmt.Sprintf("%d%%", e.ColdProgress), Description: "低体温の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModColdProgress)},
		statusItemData{Label: "高体温進行", Value: fmt.Sprintf("%d%%", e.HeatProgress), Description: "高体温の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModHeatProgress)},
		statusItemData{Label: "空腹進行", Value: fmt.Sprintf("%d%%", e.HungerProgress), Description: "空腹の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModHungerProgress)},
		statusItemData{Label: "回復効果", Value: fmt.Sprintf("%d%%", e.HealingEffect), Description: "回復アイテムの効果倍率。高いほど多く回復する", Details: sourceToDetails(e.Sources, gc.ModHealingEffect)},
	)

	// 行動
	items = append(items, statusItemData{Label: "行動", IsHeader: true, Description: "行動に関する効果"})
	items = append(items,
		statusItemData{Label: "移動速度", Value: fmt.Sprintf("%d%%", e.MoveCost), Description: "移動時のAPコスト倍率。低いほど少ないAPで移動できる", Details: sourceToDetails(e.Sources, gc.ModMoveCost)},
		statusItemData{Label: "発見", Value: fmt.Sprintf("%d%%", e.Exploration), Description: "アイテム発見率の倍率。高いほど見つけやすい", Details: sourceToDetails(e.Sources, gc.ModExploration)},
		statusItemData{Label: "被発見", Value: fmt.Sprintf("%d%%", e.EnemyVision), Description: "敵に発見される距離の倍率。低いほど見つかりにくい", Details: sourceToDetails(e.Sources, gc.ModEnemyVision)},
		statusItemData{Label: "暗所視界", Value: fmt.Sprintf("%d%%", e.NightVision), Description: "暗所での視界の倍率。高いほど見える", Details: sourceToDetails(e.Sources, gc.ModNightVision)},
	)

	// 生産
	items = append(items, statusItemData{Label: "生産", IsHeader: true, Description: "生産・取引に関する効果"})
	items = append(items,
		statusItemData{Label: "素材消費", Value: fmt.Sprintf("%d%%", e.CraftCost), Description: "合成時の素材消費量倍率。低いほど素材が節約できる", Details: sourceToDetails(e.Sources, gc.ModCraftCost)},
		statusItemData{Label: "合成品質", Value: fmt.Sprintf("%d%%", e.SmithQuality), Description: "調合時の品質倍率。高いほど良い品ができる", Details: sourceToDetails(e.Sources, gc.ModSmithQuality)},
		statusItemData{Label: "買値", Value: fmt.Sprintf("%d%%", e.BuyPrice), Description: "買い物の価格倍率。低いほど安く買える", Details: sourceToDetails(e.Sources, gc.ModBuyPrice)},
		statusItemData{Label: "売値", Value: fmt.Sprintf("%d%%", e.SellPrice), Description: "売却の価格倍率。高いほど高く売れる", Details: sourceToDetails(e.Sources, gc.ModSellPrice)},
		statusItemData{Label: "最大重量", Value: fmt.Sprintf("%d%%", e.MaxWeight), Description: "所持可能な最大重量の倍率", Details: sourceToDetails(e.Sources, gc.ModMaxWeight)},
		statusItemData{Label: "最大荷重", Value: fmt.Sprintf("%d%%", e.HeavyArmor), Description: "最大荷重倍率", Details: sourceToDetails(e.Sources, gc.ModHeavyArmor)},
	)

	return items
}

func (st *StatusState) createHealthItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := make([]statusItemData, 0, int(gc.BodyPartCount))

	var hs *gc.HealthStatus
	if aliveHas(world, world.Components.HealthStatus, playerEntity) {
		hs = world.Components.HealthStatus.Get(playerEntity)
	}

	for i := range int(gc.BodyPartCount) {
		part := gc.BodyPart(i)

		var conditionStr strings.Builder
		if hs != nil {
			conditions := hs.Parts[i].Conditions
			for j, cond := range conditions {
				if j > 0 {
					conditionStr.WriteString(", ")
				}
				conditionStr.WriteString(cond.DisplayName())
			}
		}

		value := conditionStr.String()
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
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, "status")
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	// 1列グリッドで縦に並べる。コンテンツ行だけ縦に伸縮する
	root := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(1),
				widget.GridLayoutOpts.Spacing(0, theme.Space2),
				widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, false, true, false}),
				widget.GridLayoutOpts.Padding(&widget.Insets{
					Top:    theme.Space3,
					Bottom: theme.Space3,
					Left:   theme.Space3,
					Right:  theme.Space3,
				}),
			),
		),
	)

	// Row 0: タイトル
	root.AddChild(styled.NewTitleText(props.PlayerName, res))

	// Row 1: カテゴリタブを中央寄せ
	tabRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	categoryInner := st.buildCategoryContainer(props.Tabs, tabIndex, res)
	categoryInner.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
	}
	tabRow.AddChild(categoryInner)
	root.AddChild(tabRow)

	// Row 2: コンテンツ。常に2列で半々に分割する
	content := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(2),
				widget.GridLayoutOpts.Spacing(theme.Space3, 0),
				widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{true}),
			),
		),
	)
	content.AddChild(st.buildItemContainer(props.Tabs, tabIndex, itemIndex, res))
	content.AddChild(st.buildDetailContainer(world, props, tabIndex, itemIndex, res))
	root.AddChild(content)

	// Row 3: 説明文
	root.AddChild(st.buildDescContainer(props.Tabs, tabIndex, itemIndex, res))

	return &ebitenui.UI{Container: root}
}

func (st *StatusState) buildCategoryContainer(tabs []statusTabData, tabIndex int, res resources.UIResources) *widget.Container {
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

func (st *StatusState) buildItemContainer(tabs []statusTabData, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
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

	// 能力タブは修正値列があるため3列、他のタブは2列で値を右端に寄せる
	hasModifier := currentTab.ID == tabAbilities
	var columnWidths []int
	var aligns []styled.TextAlign
	if hasModifier {
		columnWidths = []int{100, 60, 60}
		aligns = []styled.TextAlign{styled.AlignLeft, styled.AlignRight, styled.AlignRight}
	} else {
		columnWidths = []int{100, 60}
		aligns = []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	}

	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
		if entry.Item.IsHeader {
			if hasModifier {
				styled.NewTableHeaderRow(table, columnWidths, []string{entry.Item.Label, "", ""}, res)
			} else {
				styled.NewTableHeaderRow(table, columnWidths, []string{entry.Item.Label, ""}, res)
			}
			continue
		}
		isSelected := pg.IsSelectedInPage(entry.Index)
		if hasModifier {
			styled.NewTableRow(table, columnWidths, []string{entry.Item.Label, entry.Item.Value, entry.Item.Modifier}, aligns, &isSelected, res)
		} else {
			styled.NewTableRow(table, columnWidths, []string{entry.Item.Label, entry.Item.Value}, aligns, &isSelected, res)
		}
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(項目なし)", res))
	}

	return container
}

func (st *StatusState) buildDetailContainer(world w.World, props statusProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	needsDetail := false
	if tabIndex < len(props.Tabs) && itemIndex < len(props.Tabs[tabIndex].Items) {
		switch props.Tabs[tabIndex].ID {
		case tabSkills, tabEffects, tabHealth:
			needsDetail = true
		}
	}

	if !needsDetail {
		return styled.NewVerticalContainer()
	}

	container := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	item := props.Tabs[tabIndex].Items[itemIndex]
	switch props.Tabs[tabIndex].ID {
	case tabSkills:
		st.buildSkillDetail(container, item, res)
	case tabEffects:
		st.buildEffectDetail(container, item, res)
	case tabHealth:
		st.buildHealthDetail(container, world, res)
	}

	return container
}

func (st *StatusState) buildSkillDetail(container *widget.Container, item statusItemData, res resources.UIResources) {
	for _, d := range item.Details {
		container.AddChild(styled.NewDescriptionText(d.Label, res))
		container.AddChild(styled.NewMenuText(fmt.Sprintf(" %s", d.Value), res))
	}
}

func (st *StatusState) buildEffectDetail(container *widget.Container, item statusItemData, res resources.UIResources) {
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

func (st *StatusState) buildHealthDetail(container *widget.Container, world w.World, res resources.UIResources) {
	var playerEntity ecs.Entity
	query.Player(world, func(entity ecs.Entity) {
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

func (st *StatusState) buildDescContainer(tabs []statusTabData, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
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
