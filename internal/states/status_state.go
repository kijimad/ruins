package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
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

	// 環境気温の内訳
	baseTemp     int
	tileModifier int
	timeModifier int
	tileTemp     *gc.TileTemperature

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

	var err error
	st.envTemp, err = st.calculateEnvTemp(world)
	if err != nil {
		return err
	}
	st.ui = st.initUI(world)

	return nil
}

// calculateEnvTemp は環境気温を計算し、内訳を保存する
// TODO: ここに気温の計算ロジックがあるべきではない。計算結果をResourcesにキャッシュしておいたほうがいいかも
func (st *StatusState) calculateEnvTemp(world w.World) (int, error) {
	dungeonRes := world.Resources.Dungeon
	if dungeonRes == nil {
		return 0, fmt.Errorf("ダンジョンリソースがありません")
	}

	def, ok := dungeon.GetDungeon(dungeonRes.DefinitionName)
	if !ok {
		return 0, fmt.Errorf("ダンジョン定義が見つかりません: %s", dungeonRes.DefinitionName)
	}

	st.baseTemp = def.BaseTemperature

	st.timeModifier = 0
	if world.Resources.GameTime != nil {
		st.timeModifier = world.Resources.GameTime.GetTemperatureModifier()
	}

	st.tileModifier = 0
	st.tileTemp = nil
	if st.playerEntity.HasComponent(world.Components.GridElement) {
		gridElement := world.Components.GridElement.Get(st.playerEntity).(*gc.GridElement)
		st.tileModifier, st.tileTemp = st.getTileTemperature(world, gridElement.X, gridElement.Y)
	}

	return st.baseTemp + st.timeModifier + st.tileModifier, nil
}

// getTileTemperature はタイルの気温修正と詳細を取得する
func (st *StatusState) getTileTemperature(world w.World, x, y gc.Tile) (int, *gc.TileTemperature) {
	var modifier int
	var tileTemp *gc.TileTemperature
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.TileTemperature,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		if grid.X == x && grid.Y == y {
			tileTemp = world.Components.TileTemperature.Get(entity).(*gc.TileTemperature)
			modifier = tileTemp.Total()
		}
	}))
	return modifier, tileTemp
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
			st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
			st.updateCategoryDisplay(world)
		},
		OnItemChange: func(_ int, _, _ int, item tabmenu.Item) error {
			st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
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
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)

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
			ID:    "temperature",
			Label: "体温",
			Items: st.createTemperatureItems(world),
		},
	}
}

// createBasicItems は基本ステータス項目を作成する
func (st *StatusState) createBasicItems(world w.World) []tabmenu.Item {
	items := []tabmenu.Item{}

	if st.playerEntity.HasComponent(world.Components.Pools) {
		pools := world.Components.Pools.Get(st.playerEntity).(*gc.Pools)
		items = append(items,
			st.newStatusItem("HP", fmt.Sprintf("%d / %d", pools.HP.Current, pools.HP.Max), "体力。0になると死亡する"),
			st.newStatusItem("SP", fmt.Sprintf("%d / %d", pools.SP.Current, pools.SP.Max), "スタミナ。行動に消費する"),
			st.newStatusItem("EP", fmt.Sprintf("%d / %d", pools.EP.Current, pools.EP.Max), "電力。電子機器の使用に消費する"),
			st.newStatusItem("重量", fmt.Sprintf("%.1f / %.1f kg", pools.Weight.Current, pools.Weight.Max), "所持重量。超過すると移動が遅くなる"),
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
		st.newStatusItem("環境気温", fmt.Sprintf("%d°C", st.envTemp), "現在地の気温"),
	)
	if world.Resources.GameTime != nil {
		items = append(items,
			st.newStatusItem("時間帯", world.Resources.GameTime.GetTimeOfDay().String(), "現在の時間帯。屋外では気温に影響する"),
		)
	}

	return items
}

// createAttributeItems は能力値項目を作成する
func (st *StatusState) createAttributeItems(world w.World) []tabmenu.Item {
	items := []tabmenu.Item{}

	if st.playerEntity.HasComponent(world.Components.Attributes) {
		attrs := world.Components.Attributes.Get(st.playerEntity).(*gc.Attributes)
		items = append(items,
			st.newStatusItem(consts.VitalityLabel, fmt.Sprintf("%d (%+d)", attrs.Vitality.Total, attrs.Vitality.Modifier), "体力。HPとSPの最大値に影響する"),
			st.newStatusItem(consts.StrengthLabel, fmt.Sprintf("%d (%+d)", attrs.Strength.Total, attrs.Strength.Modifier), "筋力。近接攻撃のダメージに影響する"),
			st.newStatusItem(consts.SensationLabel, fmt.Sprintf("%d (%+d)", attrs.Sensation.Total, attrs.Sensation.Modifier), "感覚。射撃攻撃のダメージに影響する"),
			st.newStatusItem(consts.DexterityLabel, fmt.Sprintf("%d (%+d)", attrs.Dexterity.Total, attrs.Dexterity.Modifier), "器用さ。命中率に影響する"),
			st.newStatusItem(consts.AgilityLabel, fmt.Sprintf("%d (%+d)", attrs.Agility.Total, attrs.Agility.Modifier), "敏捷。回避率と行動速度に影響する"),
			st.newStatusItem(consts.DefenseLabel, fmt.Sprintf("%d (%+d)", attrs.Defense.Total, attrs.Defense.Modifier), "防御。被ダメージを軽減する"),
		)
	}

	return items
}

// createTemperatureItems は体温項目を作成する
func (st *StatusState) createTemperatureItems(world w.World) []tabmenu.Item {
	items := []tabmenu.Item{}

	if st.playerEntity.HasComponent(world.Components.BodyTemperature) {
		bt := world.Components.BodyTemperature.Get(st.playerEntity).(*gc.BodyTemperature)

		for i := 0; i < int(gc.BodyPartCount); i++ {
			part := gc.BodyPart(i)
			state := bt.Parts[i]
			level := bt.GetLevel(part)

			arrow := ""
			if state.Temp < state.Convergent {
				arrow = "↑"
			} else if state.Temp > state.Convergent {
				arrow = "↓"
			}

			frostbite := ""
			if state.HasFrostbite && gc.IsExtremity(part) {
				frostbite = "[凍傷]"
			}

			gauge := st.tempGauge(state.Temp)
			value := fmt.Sprintf("%s %s%s%s", gauge, level.String(), frostbite, arrow)
			desc := st.getBodyPartDescription(part, level, state)

			items = append(items, st.newTemperatureItem(part, value, desc))
		}
	}

	return items
}

// tempGauge は温度値をテキストゲージに変換する
// 0-100の温度を10ブロックのゲージで表現する
func (st *StatusState) tempGauge(temp int) string {
	const gaugeWidth = 10
	filled := temp * gaugeWidth / 100
	if filled < 0 {
		filled = 0
	}
	if filled > gaugeWidth {
		filled = gaugeWidth
	}

	gauge := "["
	for i := 0; i < gaugeWidth; i++ {
		if i < filled {
			gauge += "#"
		} else {
			gauge += "-"
		}
	}
	gauge += "]"
	return gauge
}

// getBodyPartDescription は部位の説明を返す
func (st *StatusState) getBodyPartDescription(part gc.BodyPart, level gc.TempLevel, state gc.BodyPartState) string {
	result := ""

	switch level {
	case gc.TempLevelFreezing:
		result = "ペナルティ: -3"
	case gc.TempLevelVeryCold:
		result = "ペナルティ: -2"
	case gc.TempLevelCold:
		result = "ペナルティ: -1"
	case gc.TempLevelNormal:
		// ペナルティなし
	case gc.TempLevelHot:
		result = "ペナルティ: -1"
	case gc.TempLevelVeryHot:
		result = "ペナルティ: -2"
	case gc.TempLevelScorching:
		result = "ペナルティ: -3"
	}

	if gc.IsExtremity(part) {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("凍傷リスク: %d%%", state.FrostbiteTimer)
	}

	if result == "" {
		return " "
	}
	return result
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

// newTemperatureItem は体温項目を作成する
func (st *StatusState) newTemperatureItem(part gc.BodyPart, value, description string) tabmenu.Item {
	return tabmenu.Item{
		ID:               part.String(),
		Label:            part.String(),
		AdditionalLabels: []string{value},
		UserData: statusItem{
			Label:       part.String(),
			Value:       value,
			Description: description,
			TabID:       "temperature",
			BodyPart:    part,
		},
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

	// 体温タブの場合は内訳を右側に表示
	if data.TabID == "temperature" {
		st.addTemperatureBreakdown(world, data.BodyPart)
	}
}

// addTemperatureBreakdown は体温の内訳を詳細コンテナに追加する
func (st *StatusState) addTemperatureBreakdown(world w.World, part gc.BodyPart) {
	res := world.Resources.UIResources

	// 区切り線代わりの空行
	st.detailContainer.AddChild(st.newDetailText(" ", res))
	st.detailContainer.AddChild(st.newDetailText("環境", res))

	// 気温
	st.detailContainer.AddChild(st.newDetailRow("気温", fmt.Sprintf("%dC", st.baseTemp), res))

	// タイル修正の内訳
	if st.tileTemp != nil {
		if st.tileTemp.Shelter != gc.ShelterNone {
			shelterLabel := "屋内"
			if st.tileTemp.Shelter == gc.ShelterPartial {
				shelterLabel = "半屋外"
			}
			st.detailContainer.AddChild(st.newDetailRow(shelterLabel, fmt.Sprintf("%+dC", st.tileTemp.Shelter), res))
		}
		if st.tileTemp.Water != gc.WaterNone {
			waterLabel := "水辺"
			if st.tileTemp.Water == gc.WaterSubmerged {
				waterLabel = "水中"
			}
			st.detailContainer.AddChild(st.newDetailRow(waterLabel, fmt.Sprintf("%+dC", st.tileTemp.Water), res))
		}
		if st.tileTemp.Foliage != gc.FoliageNone {
			foliageLabel := "草原"
			if st.tileTemp.Foliage == gc.FoliageForest {
				foliageLabel = "森"
			}
			st.detailContainer.AddChild(st.newDetailRow(foliageLabel, fmt.Sprintf("%+dC", st.tileTemp.Foliage), res))
		}
	}

	// 時間帯修正（常に表示）
	if world.Resources.GameTime != nil {
		timeLabel := world.Resources.GameTime.GetTimeOfDay().String()
		st.detailContainer.AddChild(st.newDetailRow(timeLabel, fmt.Sprintf("%+dC", st.timeModifier), res))
	}

	// 合計環境気温
	st.detailContainer.AddChild(st.newDetailRow("合計", fmt.Sprintf("%dC", st.envTemp), res))

	// 装備保温値
	st.detailContainer.AddChild(st.newDetailText(" ", res))
	st.detailContainer.AddChild(st.newDetailText("装備", res))

	warmth := 0
	if st.playerEntity.HasComponent(world.Components.BodyTemperature) {
		bt := world.Components.BodyTemperature.Get(st.playerEntity).(*gc.BodyTemperature)
		warmth = bt.EquippedWarmth[part]
	}
	st.detailContainer.AddChild(st.newDetailRow("保温値", fmt.Sprintf("%+d", warmth), res))
}

// newDetailText は詳細表示用の明るいテキストを作成する
func (st *StatusState) newDetailText(text string, res *resources.UIResources) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(text, &res.Text.BodyFace, consts.TextColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
		),
	)
}

// newDetailRow は詳細パネル用のラベル・値ペアを作成する
func (st *StatusState) newDetailRow(label, value string, res *resources.UIResources) *widget.Container {
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)

	// ラベル（固定幅80px）
	labelText := widget.NewText(
		widget.TextOpts.Text(label, &res.Text.BodyFace, consts.TextColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
			widget.WidgetOpts.MinSize(80, 0),
		),
	)
	container.AddChild(labelText)

	// 値
	valueText := widget.NewText(
		widget.TextOpts.Text(value, &res.Text.BodyFace, consts.TextColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{}),
		),
	)
	container.AddChild(valueText)

	return container
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
