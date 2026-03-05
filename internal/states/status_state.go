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
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// StatusState はステータス画面のステート
type StatusState struct {
	es.BaseState[w.World]
	ui           *ebitenui.UI
	playerEntity ecs.Entity
	envTemp      int

	menuView            *tabmenu.View
	itemDesc            *widget.Text      // 項目の説明（下部に表示）
	detailContainer     *widget.Container // 詳細表示のコンテナ（右側に表示）
	rootContainer       *widget.Container
	tabDisplayContainer *widget.Container
	categoryContainer   *widget.Container
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

// OnStart はステートが開始される際に呼ばれる
func (st *StatusState) OnStart(world w.World) error {
	var found bool
	worldhelper.QueryPlayer(world, func(entity ecs.Entity) {
		if !found {
			st.playerEntity = entity
			found = true
		}
	})

	if !found {
		return fmt.Errorf("プレイヤーが見つかりません")
	}

	// プレイヤー位置の環境気温を計算
	if st.playerEntity.HasComponent(world.Components.GridElement) {
		gridElement := world.Components.GridElement.Get(st.playerEntity).(*gc.GridElement)
		var err error
		st.envTemp, err = systems.CalculateEnvTemperature(world, gridElement.X, gridElement.Y)
		if err != nil {
			return err
		}
	}
	st.ui = st.initUI(world)

	return nil
}

// OnStop はステートが終了する際に呼ばれる
func (st *StatusState) OnStop(_ w.World) error { return nil }

// Update はステートの更新処理
func (st *StatusState) Update(world w.World) (es.Transition[w.World], error) {
	action, ok := st.HandleInput(world.Config)
	if ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	if err := st.menuView.Update(); err != nil {
		return es.Transition[w.World]{}, err
	}
	st.ui.Update()

	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理
func (st *StatusState) Draw(_ w.World, screen *ebiten.Image) error {
	st.ui.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *StatusState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}
	return "", false
}

// DoAction はActionを実行する
func (st *StatusState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// initUI はUIを初期化する
func (st *StatusState) initUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources

	// TabMenuの設定
	tabs := st.createTabs(world)
	config := tabmenu.Config{
		Tabs:             tabs,
		InitialTabIndex:  0,
		InitialItemIndex: 0,
		WrapNavigation:   true,
		ItemsPerPage:     20,
	}

	callbacks := tabmenu.Callbacks{
		OnSelectItem: func(_ int, _ int, _ tabmenu.TabItem, _ tabmenu.Item) error {
			return nil // ステータス画面では選択アクションなし
		},
		OnCancel: func() {
			st.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		},
		OnTabChange: func(_, _ int, _ tabmenu.TabItem) {
			st.updateTabDisplayAsTable(world)
			st.updateCategoryDisplay(world)
		},
		OnItemChange: func(_ int, _, _ int, item tabmenu.Item) error {
			st.updateTabDisplayAsTable(world)
			st.updateDetailContainer(world, item)
			return nil
		},
	}

	st.menuView = tabmenu.NewView(config, callbacks, world)

	// 説明文（下部に表示）
	itemDescContainer := styled.NewRowContainer()
	st.itemDesc = styled.NewMenuText(" ", res)
	itemDescContainer.AddChild(st.itemDesc)

	// 詳細表示コンテナ（右側に表示）
	st.detailContainer = styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// タブ表示のコンテナ
	st.tabDisplayContainer = styled.NewVerticalContainer()
	st.updateTabDisplayAsTable(world)

	// カテゴリ一覧のコンテナ
	st.categoryContainer = styled.NewRowContainer()
	st.updateCategoryDisplay(world)

	// 初期表示を更新
	st.updateInitialDisplay(world)

	st.rootContainer = styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	{
		// 3x3グリッドレイアウト
		// 1行目
		st.rootContainer.AddChild(styled.NewTitleText("ステータス", res))
		st.rootContainer.AddChild(st.categoryContainer)
		st.rootContainer.AddChild(widget.NewContainer())

		// 2行目
		st.rootContainer.AddChild(st.tabDisplayContainer)
		st.rootContainer.AddChild(widget.NewContainer())
		st.rootContainer.AddChild(st.detailContainer)

		// 3行目
		st.rootContainer.AddChild(itemDescContainer)
		st.rootContainer.AddChild(widget.NewContainer())
		st.rootContainer.AddChild(widget.NewContainer())
	}

	return &ebitenui.UI{Container: st.rootContainer}
}

// statusItem はステータス項目のデータ
type statusItem struct {
	Label       string
	Value       string
	Modifier    string // 補正値（能力値タブで使用）
	Description string
	TabID       string      // タブID（詳細表示の切り替えに使用）
	BodyPart    gc.BodyPart // 体温タブの場合、部位を指定
}

// createTabs はタブを作成する
func (st *StatusState) createTabs(world w.World) []tabmenu.TabItem {
	return []tabmenu.TabItem{
		{
			ID:    "basic",
			Label: "基本",
			Items: st.createBasicItems(world),
		},
		{
			ID:    "attributes",
			Label: "能力",
			Items: st.createAttributeItems(world),
		},
		{
			ID:    "health",
			Label: "健康",
			Items: st.createHealthItems(world),
		},
	}
}

// createBasicItems は基本ステータス項目を作成する
func (st *StatusState) createBasicItems(world w.World) []tabmenu.Item {
	items := []tabmenu.Item{}

	if st.playerEntity.HasComponent(world.Components.Pools) {
		pools := world.Components.Pools.Get(st.playerEntity).(*gc.Pools)
		items = append(items,
			st.newStatusItem("HP", fmt.Sprintf("%d", pools.HP.Max), "体力。0になると死亡する"),
			st.newStatusItem("SP", fmt.Sprintf("%d", pools.SP.Max), "スタミナ。行動に消費する"),
			st.newStatusItem("EP", fmt.Sprintf("%d", pools.EP.Max), "電力。電子機器の使用に消費する"),
			st.newStatusItem("最大重量", fmt.Sprintf("%.1fkg", pools.Weight.Max), "所持可能な最大重量"),
		)
	}

	if st.playerEntity.HasComponent(world.Components.Hunger) {
		hunger := world.Components.Hunger.Get(st.playerEntity).(*gc.Hunger)
		items = append(items,
			st.newStatusItem("空腹度", fmt.Sprintf("%d (%s)", hunger.Current, hunger.GetLevel().String()), "空腹度。高いと行動に支障が出る"),
		)
	}

	// 環境情報
	items = append(items,
		st.newStatusItem("環境気温", fmt.Sprintf("%d%s", st.envTemp, consts.IconDegree), "現在地の気温"),
	)
	items = append(items,
		st.newStatusItem("時間帯", world.Resources.Dungeon.GameTime.GetTimeOfDay().String(), "現在の時間帯。屋外では気温に影響する"),
	)

	return items
}

// createAttributeItems は能力値項目を作成する
func (st *StatusState) createAttributeItems(world w.World) []tabmenu.Item {
	items := []tabmenu.Item{}

	if st.playerEntity.HasComponent(world.Components.Attributes) {
		attrs := world.Components.Attributes.Get(st.playerEntity).(*gc.Attributes)
		items = append(items,
			st.newAttributeItem(consts.VitalityLabel, attrs.Vitality.Total, attrs.Vitality.Modifier, "体力。HPとSPの最大値に影響する"),
			st.newAttributeItem(consts.StrengthLabel, attrs.Strength.Total, attrs.Strength.Modifier, "筋力。近接攻撃のダメージに影響する"),
			st.newAttributeItem(consts.SensationLabel, attrs.Sensation.Total, attrs.Sensation.Modifier, "感覚。射撃攻撃のダメージに影響する"),
			st.newAttributeItem(consts.DexterityLabel, attrs.Dexterity.Total, attrs.Dexterity.Modifier, "器用さ。命中率に影響する"),
			st.newAttributeItem(consts.AgilityLabel, attrs.Agility.Total, attrs.Agility.Modifier, "敏捷。回避率と行動速度に影響する"),
			st.newAttributeItem(consts.DefenseLabel, attrs.Defense.Total, attrs.Defense.Modifier, "防御。被ダメージを軽減する"),
		)
	}

	return items
}

// createHealthItems は健康状態項目を作成する
func (st *StatusState) createHealthItems(world w.World) []tabmenu.Item {
	items := []tabmenu.Item{}

	// HealthStatus から各部位の状態を取得する
	var hs *gc.HealthStatus
	if st.playerEntity.HasComponent(world.Components.HealthStatus) {
		hs = world.Components.HealthStatus.Get(st.playerEntity).(*gc.HealthStatus)
	}

	for i := 0; i < int(gc.BodyPartCount); i++ {
		part := gc.BodyPart(i)

		// 状態の文字列を構築
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

		// 状態がない場合は「正常」と表示
		value := conditionStr
		if value == "" {
			value = "正常"
		}
		desc := st.getHealthPartDescription(world, part)

		items = append(items, st.newHealthItem(part, value, desc))
	}

	return items
}

// newStatusItem はステータス項目を作成する
func (st *StatusState) newStatusItem(label, value, description string) tabmenu.Item {
	return tabmenu.Item{
		ID:               label,
		Label:            label,
		AdditionalLabels: []string{value},
		UserData: statusItem{
			Label:       label,
			Value:       value,
			Description: description,
			TabID:       "basic",
		},
	}
}

// newAttributeItem は能力値項目を作成する
func (st *StatusState) newAttributeItem(label string, total, modifier int, description string) tabmenu.Item {
	return tabmenu.Item{
		ID:               label,
		Label:            label,
		AdditionalLabels: []string{fmt.Sprintf("%d", total)},
		UserData: statusItem{
			Label:       label,
			Value:       fmt.Sprintf("%d", total),
			Modifier:    fmt.Sprintf("(%+d)", modifier),
			Description: description,
			TabID:       "attributes",
		},
	}
}

// newHealthItem は健康状態項目を作成する
func (st *StatusState) newHealthItem(part gc.BodyPart, value, description string) tabmenu.Item {
	return tabmenu.Item{
		ID:               part.String(),
		Label:            part.String(),
		AdditionalLabels: []string{value},
		UserData: statusItem{
			Label:       part.String(),
			Value:       value,
			Description: description,
			TabID:       "health",
			BodyPart:    part,
		},
	}
}

// getHealthPartDescription は部位の説明を返す
func (st *StatusState) getHealthPartDescription(_ w.World, part gc.BodyPart) string {
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
	case gc.BodyPartCount:
		panic("BodyPartCountは有効な部位ではない")
	default:
		panic("不正なBodyPart値")
	}
}

// updateDetailContainer は詳細コンテナと説明文を更新する
func (st *StatusState) updateDetailContainer(world w.World, item tabmenu.Item) {
	st.detailContainer.RemoveChildren()

	if item.UserData == nil {
		st.itemDesc.Label = " "
		return
	}

	data, ok := item.UserData.(statusItem)
	if !ok {
		st.itemDesc.Label = " "
		return
	}

	// 説明テキストを下部に表示
	if data.Description != "" {
		st.itemDesc.Label = data.Description
	} else {
		st.itemDesc.Label = " "
	}

	// タブに応じて右側の内訳を表示
	if data.TabID == "health" {
		st.addHealthBreakdown(world, data.BodyPart)
	}
}

// addHealthBreakdown は選択中の部位の健康状態を詳細コンテナに追加する
func (st *StatusState) addHealthBreakdown(world w.World, part gc.BodyPart) {
	res := world.Resources.UIResources
	columnWidths := []int{80, 100}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight}

	// 快適温度範囲を表示
	allInsulation := systems.CalculateEquippedInsulation(world, st.playerEntity)
	lowerBound, upperBound := systems.ComfortableRange(allInsulation[part])

	tempTable := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(tempTable, columnWidths, []string{"快適温度", ""}, res)
	styled.NewTableRow(tempTable, columnWidths, []string{"範囲", fmt.Sprintf("%d%s 〜 %d%s", lowerBound, consts.IconDegree, upperBound, consts.IconDegree)}, aligns, nil, res)
	st.detailContainer.AddChild(tempTable)

	// 健康状態
	if !st.playerEntity.HasComponent(world.Components.HealthStatus) {
		return
	}

	hs := world.Components.HealthStatus.Get(st.playerEntity).(*gc.HealthStatus)
	partHealth := hs.Parts[part]

	statusTable := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(statusTable, columnWidths, []string{"状態", ""}, res)

	if len(partHealth.Conditions) == 0 {
		styled.NewTableRow(statusTable, columnWidths, []string{"", "正常"}, aligns, nil, res)
		st.detailContainer.AddChild(statusTable)
		return
	}

	// 各状態とその影響を表示
	for _, cond := range partHealth.Conditions {
		// 状態名とタイマー進行度を表示
		condName := fmt.Sprintf("%s [%.0f%%]", cond.DisplayName(), cond.Timer)
		styled.NewTableRow(statusTable, columnWidths, []string{condName, ""}, aligns, nil, res)
		for _, effect := range cond.Effects {
			statName := effect.Stat.String()
			valueStr := fmt.Sprintf("%+d", effect.Value)
			styled.NewTableRow(statusTable, columnWidths, []string{"  " + statName, valueStr}, aligns, nil, res)
		}
	}
	st.detailContainer.AddChild(statusTable)
}

// updateTabDisplayAsTable はタブ表示コンテナをテーブル形式で更新する
func (st *StatusState) updateTabDisplayAsTable(world w.World) {
	st.tabDisplayContainer.RemoveChildren()
	res := world.Resources.UIResources

	currentTab := st.menuView.GetCurrentTab()
	currentItemIndex := st.menuView.GetCurrentItemIndex()

	// カーソル、ラベル、値、補正の4列
	columnWidths := []int{20, 100, 60, 60}
	// 値と補正は右揃え
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight, styled.AlignRight}

	table := styled.NewTableContainer(columnWidths, res)

	for i, item := range currentTab.Items {
		isSelected := i == currentItemIndex

		// UserDataからstatusItemを取得して値を表示
		data, ok := item.UserData.(statusItem)
		value := ""
		modifier := ""
		if ok {
			value = data.Value
			modifier = data.Modifier
		}

		// カーソル用の空文字、ラベル、値、補正
		styled.NewTableRow(table, columnWidths, []string{"", item.Label, value, modifier}, aligns, &isSelected, res)
	}

	st.tabDisplayContainer.AddChild(table)

	// アイテムがない場合の表示
	if len(currentTab.Items) == 0 {
		emptyText := styled.NewDescriptionText("(項目なし)", res)
		st.tabDisplayContainer.AddChild(emptyText)
	}
}

// updateCategoryDisplay はカテゴリ表示を更新する
func (st *StatusState) updateCategoryDisplay(world w.World) {
	st.categoryContainer.RemoveChildren()

	currentTabIndex := st.menuView.GetCurrentTabIndex()
	tabs := st.createTabs(world)

	for i, tab := range tabs {
		isSelected := i == currentTabIndex
		if isSelected {
			categoryWidget := styled.NewListItemText(tab.Label, consts.TextColor, true, world.Resources.UIResources)
			st.categoryContainer.AddChild(categoryWidget)
		} else {
			categoryWidget := styled.NewListItemText(tab.Label, consts.ForegroundColor, false, world.Resources.UIResources)
			st.categoryContainer.AddChild(categoryWidget)
		}
	}
}

// updateInitialDisplay は初期表示を更新する
func (st *StatusState) updateInitialDisplay(world w.World) {
	currentTab := st.menuView.GetCurrentTab()
	currentItemIndex := st.menuView.GetCurrentItemIndex()

	if len(currentTab.Items) > 0 && currentItemIndex >= 0 && currentItemIndex < len(currentTab.Items) {
		currentItem := currentTab.Items[currentItemIndex]
		st.updateDetailContainer(world, currentItem)
	}
}

// NewStatusState はステータス画面のStateを作成するファクトリー関数
func NewStatusState() es.State[w.World] {
	return &StatusState{}
}
