package states

import (
	"fmt"
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// EquipMenuState は装備メニューのゲームステート
type EquipMenuState struct {
	es.BaseState[w.World]
	ui *ebitenui.UI

	menuView            *tabmenu.View
	itemDesc            *widget.Text      // アイテムの説明
	specContainer       *widget.Container // 性能コンテナ
	abilityContainer    *widget.Container // プレイヤーの能力表示コンテナ
	rootContainer       *widget.Container
	tabDisplayContainer *widget.Container // タブ表示のコンテナ

	// 現在のタブインデックス（装備選択時の復元用）
	previousTabIndex int

	// アクション選択ウィンドウ用
	actionWindow     *widget.Window // アクション選択ウィンドウ
	actionFocusIndex int            // アクションウィンドウ内のフォーカス
	actionItems      []string       // アクション項目リスト
	isWindowMode     bool           // ウィンドウ操作モードかどうか

	// 装備選択状態管理
	isEquipMode       bool                   // 装備選択モードかどうか
	equipSlotNumber   gc.EquipmentSlotNumber // 装備スロット番号
	previousEquipment *ecs.Entity            // 前の装備
	equipTargetMember ecs.Entity             // 装備対象のメンバー
}

func (st EquipMenuState) String() string {
	return "EquipMenu"
}

// State interface ================

var _ es.State[w.World] = &EquipMenuState{}
var _ es.ActionHandler[w.World] = &EquipMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *EquipMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *EquipMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *EquipMenuState) OnStart(world w.World) error {
	st.ui = st.initUI(world)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *EquipMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *EquipMenuState) Update(world w.World) (es.Transition[w.World], error) {
	for _, updater := range []w.Updater{
		&gs.EquipmentChangedSystem{},
		&gs.InventoryChangedSystem{},
	} {
		changed := false

		if sys, ok := world.Updaters[updater.String()]; ok {
			if shouldRunner, ok := sys.(gs.ShouldRunner); ok {
				if shouldRunner.ShouldRun(world) {
					changed = true
				}
			}

			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}

			if changed {
				st.reloadAbilityContainer(world)
			}
		}
	}

	// キー入力をActionに変換
	var action inputmapper.ActionID
	var ok bool
	if st.isWindowMode {
		action, ok = HandleWindowInput()
	} else {
		action, ok = st.HandleInput()
	}

	if ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	// アクションウィンドウ表示中はTabMenuの更新をスキップ
	if !st.isWindowMode {
		if err := st.menuView.Update(); err != nil {
			return es.Transition[w.World]{}, err
		}
	}
	st.ui.Update()

	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *EquipMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.ui.Draw(screen)
	return nil
}

// ================

// HandleInput はキー入力をActionに変換する
func (st *EquipMenuState) HandleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}

	return "", false
}

// DoAction はActionを実行する
func (st *EquipMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	// ウィンドウモード時のアクション処理
	if st.isWindowMode {
		switch action {
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			if UpdateFocusIndex(action, &st.actionFocusIndex, len(st.actionItems)) {
				st.updateActionWindowDisplay(world)
			}
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowConfirm:
			st.executeActionItem(world)
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowCancel:
			st.closeActionWindow()
			return es.Transition[w.World]{Type: es.TransNone}, nil
		default:
			return es.Transition[w.World]{}, fmt.Errorf("ウィンドウモード時の未知のアクション: %s", action)
		}
	}

	switch action {
	case inputmapper.ActionOpenDebugMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// ================

func (st *EquipMenuState) initUI(world w.World) *ebitenui.UI {
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
		OnSelectItem: func(_ int, _ int, tab tabmenu.TabItem, item tabmenu.Item) error {
			return st.handleItemSelection(world, tab, item)
		},
		OnCancel: func() {
			// Escapeで前の画面に戻る
			st.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		},
		OnTabChange: func(_, _ int, _ tabmenu.TabItem) {
			st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
			st.reloadAbilityContainer(world)
		},
		OnItemChange: func(_ int, _, _ int, item tabmenu.Item) error {
			if err := st.handleItemChange(world, item); err != nil {
				return err
			}
			st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
			return nil
		},
	}

	st.menuView = tabmenu.NewView(config, callbacks, world)

	// アイテムの説明文
	itemDescContainer := styled.NewRowContainer()
	st.itemDesc = styled.NewMenuText(" ", world.Resources.UIResources) // 空文字だと初期状態の縦サイズがなくなる
	itemDescContainer.AddChild(st.itemDesc)

	st.specContainer = styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	st.abilityContainer = styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 初期状態の表示を更新
	if err := st.updateInitialItemDisplay(world); err != nil {
		log.Fatalf("Failed to update initial item display: %v", err)
	}

	// タブ表示のコンテナを作成
	st.tabDisplayContainer = styled.NewVerticalContainer()
	st.createTabDisplayUI(world)

	st.rootContainer = styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	{
		// 3x3グリッドレイアウト: 9個の要素が必要
		// 1行目
		st.rootContainer.AddChild(styled.NewTitleText("装備", world.Resources.UIResources))
		st.rootContainer.AddChild(widget.NewContainer()) // 空
		st.rootContainer.AddChild(widget.NewContainer()) // 空

		// 2行目
		st.rootContainer.AddChild(st.tabDisplayContainer) // タブとアイテム一覧の表示
		st.rootContainer.AddChild(widget.NewContainer())  // 空
		st.rootContainer.AddChild(styled.NewWSplitContainer(st.specContainer, st.abilityContainer))

		// 3行目
		st.rootContainer.AddChild(itemDescContainer)
		st.rootContainer.AddChild(widget.NewContainer()) // 空
		st.rootContainer.AddChild(widget.NewContainer()) // 空
	}

	return &ebitenui.UI{Container: st.rootContainer}
}

// createTabs はTabMenuで使用するタブを作成する
func (st *EquipMenuState) createTabs(world w.World) []tabmenu.TabItem {
	var player ecs.Entity
	var found bool
	worldhelper.QueryPlayer(world, func(entity ecs.Entity) {
		if !found {
			player = entity
			found = true
		}
	})

	if !found {
		return []tabmenu.TabItem{}
	}

	allItems := st.createAllSlotItems(world, player, 0)
	tabs := []tabmenu.TabItem{
		{
			ID:    "player_equipment",
			Label: "装備",
			Items: allItems,
		},
	}

	return tabs
}

// createAllSlotItems は武器と防具の全スロットのMenuItemを作成する
func (st *EquipMenuState) createAllSlotItems(world w.World, member ecs.Entity, _ int) []tabmenu.Item {
	items := []tabmenu.Item{}

	// 武器スロットを追加（近接武器、遠距離武器）
	weaponSlots := []struct {
		entity     *ecs.Entity
		label      string
		slotNumber gc.EquipmentSlotNumber
	}{
		{worldhelper.GetMeleeWeapon(world, member), "近接武器", gc.SlotMeleeWeapon},
		{worldhelper.GetRangedWeapon(world, member), "遠距離武器", gc.SlotRangedWeapon},
	}
	for i, slot := range weaponSlots {
		var name string
		if slot.entity != nil {
			name = fmt.Sprintf("%s: %s", slot.label, world.Components.Name.Get(*slot.entity).(*gc.Name).Name)
		} else {
			name = fmt.Sprintf("%s: -", slot.label)
		}

		items = append(items, tabmenu.Item{
			ID:    fmt.Sprintf("weapon_slot_%d", i),
			Label: name,
			UserData: map[string]interface{}{
				"member":     member,
				"slotNumber": slot.slotNumber,
				"entity":     slot.entity,
			},
		})
	}

	// 防具スロットを追加
	armorSlots := worldhelper.GetArmorEquipments(world, member)
	armorLabels := []string{"防具1", "防具2", "防具3", "防具4"}
	armorSlotNumbers := []gc.EquipmentSlotNumber{gc.SlotHead, gc.SlotTorso, gc.SlotLegs, gc.SlotJewelry}
	for i, slot := range armorSlots {
		var name string
		if slot != nil {
			name = fmt.Sprintf("%s: %s", armorLabels[i], world.Components.Name.Get(*slot).(*gc.Name).Name)
		} else {
			name = fmt.Sprintf("%s: -", armorLabels[i])
		}

		items = append(items, tabmenu.Item{
			ID:    fmt.Sprintf("wear_slot_%d", i),
			Label: name,
			UserData: map[string]interface{}{
				"member":     member,
				"slotNumber": armorSlotNumbers[i],
				"entity":     slot,
			},
		})
	}

	return items
}

// handleItemSelection はアイテム選択時の処理
func (st *EquipMenuState) handleItemSelection(world w.World, _ tabmenu.TabItem, item tabmenu.Item) error {
	if st.isEquipMode {
		// 装備選択モードの場合
		return st.handleEquipItemSelection(world, item)
	}

	// スロット選択モードの場合
	userData, ok := item.UserData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected item UserData")
	}

	st.showActionWindow(world, userData)
	return nil
}

// handleItemChange はアイテム変更時の処理（カーソル移動）
func (st *EquipMenuState) handleItemChange(world w.World, item tabmenu.Item) error {
	// 無効なアイテムの場合は何もしない
	if item.UserData == nil {
		st.itemDesc.Label = " "
		st.specContainer.RemoveChildren()
		return nil
	}

	if st.isEquipMode {
		// 装備選択モードの場合
		entity, ok := item.UserData.(ecs.Entity)
		if !ok {
			return fmt.Errorf("unexpected item UserData")
		}

		if entity.HasComponent(world.Components.Description) {
			desc := world.Components.Description.Get(entity).(*gc.Description)
			st.itemDesc.Label = desc.Description
		}
		views.UpdateSpec(world, st.specContainer, entity)
	} else {
		// スロット選択モードの場合
		userData, ok := item.UserData.(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected item UserData")
		}

		slotEntity := userData["entity"].(*ecs.Entity)
		if slotEntity != nil {
			if (*slotEntity).HasComponent(world.Components.Description) {
				desc := world.Components.Description.Get(*slotEntity).(*gc.Description)
				st.itemDesc.Label = desc.Description
			}
			views.UpdateSpec(world, st.specContainer, *slotEntity)
		} else {
			st.itemDesc.Label = " "
			st.specContainer.RemoveChildren()
		}

		// プレイヤー情報を更新
		if _, ok := userData["member"].(ecs.Entity); ok {
			st.reloadAbilityContainer(world)
		}
	}
	return nil
}

// createTabDisplayUI はタブ表示UIを作成する
func (st *EquipMenuState) createTabDisplayUI(_ w.World) {
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
}

// updateInitialItemDisplay は初期状態のアイテム表示を更新する
func (st *EquipMenuState) updateInitialItemDisplay(world w.World) error {
	currentTab := st.menuView.GetCurrentTab()
	currentItemIndex := st.menuView.GetCurrentItemIndex()

	if len(currentTab.Items) > 0 && currentItemIndex >= 0 && currentItemIndex < len(currentTab.Items) {
		currentItem := currentTab.Items[currentItemIndex]
		if err := st.handleItemChange(world, currentItem); err != nil {
			return err
		}
	}
	return nil
}

// reloadAbilityContainer はメンバーの能力表示コンテナを更新する
func (st *EquipMenuState) reloadAbilityContainer(world w.World) {
	st.abilityContainer.RemoveChildren()

	var player ecs.Entity
	var found bool
	worldhelper.QueryPlayer(world, func(entity ecs.Entity) {
		if !found {
			player = entity
			found = true
		}
	})

	if !found {
		return
	}

	// プレイヤーの基本情報を表示
	views.AddMemberStatusText(st.abilityContainer, player, world)

	attrs := world.Components.Attributes.Get(player).(*gc.Attributes)
	st.abilityContainer.AddChild(styled.NewBodyText(fmt.Sprintf("%s %2d(%+d)", consts.VitalityLabel, attrs.Vitality.Total, attrs.Vitality.Modifier), consts.TextColor, world.Resources.UIResources))
	st.abilityContainer.AddChild(styled.NewBodyText(fmt.Sprintf("%s %2d(%+d)", consts.StrengthLabel, attrs.Strength.Total, attrs.Strength.Modifier), consts.TextColor, world.Resources.UIResources))
	st.abilityContainer.AddChild(styled.NewBodyText(fmt.Sprintf("%s %2d(%+d)", consts.SensationLabel, attrs.Sensation.Total, attrs.Sensation.Modifier), consts.TextColor, world.Resources.UIResources))
	st.abilityContainer.AddChild(styled.NewBodyText(fmt.Sprintf("%s %2d(%+d)", consts.DexterityLabel, attrs.Dexterity.Total, attrs.Dexterity.Modifier), consts.TextColor, world.Resources.UIResources))
	st.abilityContainer.AddChild(styled.NewBodyText(fmt.Sprintf("%s %2d(%+d)", consts.AgilityLabel, attrs.Agility.Total, attrs.Agility.Modifier), consts.TextColor, world.Resources.UIResources))
	st.abilityContainer.AddChild(styled.NewBodyText(fmt.Sprintf("%s %2d(%+d)", consts.DefenseLabel, attrs.Defense.Total, attrs.Defense.Modifier), consts.TextColor, world.Resources.UIResources))
}

// queryEquipableItemsForSlot はスロット番号に応じた装備可能なアイテムを取得する
func (st *EquipMenuState) queryEquipableItemsForSlot(world w.World, slotNumber gc.EquipmentSlotNumber) []ecs.Entity {
	items := []ecs.Entity{}

	// 武器スロットの場合
	if slotNumber == gc.SlotMeleeWeapon || slotNumber == gc.SlotRangedWeapon {
		world.Manager.Join(
			world.Components.Item,
			world.Components.ItemLocationInPlayerBackpack,
			world.Components.Weapon,
			world.Components.Attack,
		).Visit(ecs.Visit(func(entity ecs.Entity) {
			attack := world.Components.Attack.Get(entity).(*gc.Attack)
			if attack != nil {
				// スロット番号に応じた武器種別のみ追加
				if slotNumber == gc.SlotMeleeWeapon && attack.AttackCategory.Range == gc.AttackRangeMelee {
					items = append(items, entity)
				} else if slotNumber == gc.SlotRangedWeapon && attack.AttackCategory.Range == gc.AttackRangeRanged {
					items = append(items, entity)
				}
			}
		}))
	} else {
		// 防具スロットの場合
		// スロット番号からEquipmentTypeを決定
		var targetCategory gc.EquipmentType
		switch slotNumber {
		case gc.SlotHead:
			targetCategory = gc.EquipmentHead
		case gc.SlotTorso:
			targetCategory = gc.EquipmentTorso
		case gc.SlotLegs:
			targetCategory = gc.EquipmentLegs
		case gc.SlotJewelry:
			targetCategory = gc.EquipmentJewelry
		default:
			return worldhelper.SortEntities(world, items)
		}

		world.Manager.Join(
			world.Components.Item,
			world.Components.ItemLocationInPlayerBackpack,
			world.Components.Wearable,
		).Visit(ecs.Visit(func(entity ecs.Entity) {
			wearable := world.Components.Wearable.Get(entity).(*gc.Wearable)
			if wearable != nil && wearable.EquipmentCategory == targetCategory {
				items = append(items, entity)
			}
		}))
	}

	return worldhelper.SortEntities(world, items)
}

// showActionWindow はアクションウィンドウを表示する
func (st *EquipMenuState) showActionWindow(world w.World, userData map[string]interface{}) {
	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", world.Resources.UIResources)
	st.actionWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// アクション項目を準備
	st.actionItems = []string{}

	// スロットに装備されているかチェック
	slotEntity, hasEquipment := userData["entity"].(*ecs.Entity)
	if hasEquipment && slotEntity != nil {
		st.actionItems = append(st.actionItems, "外す")
	}
	st.actionItems = append(st.actionItems, "装備する")
	st.actionItems = append(st.actionItems, TextClose)

	st.actionFocusIndex = 0
	st.isWindowMode = true

	// UI要素を作成（表示のみ、操作はキーボードで行う）
	st.createActionWindowUI(world, windowContainer)

	st.actionWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.actionWindow)
}

// createActionWindowUI はアクションウィンドウのUI要素を作成する
func (st *EquipMenuState) createActionWindowUI(world w.World, _ *widget.Container) {
	st.updateActionWindowDisplay(world)
}

// updateActionWindowDisplay はアクションウィンドウの表示を更新する
func (st *EquipMenuState) updateActionWindowDisplay(world w.World) {
	if st.actionWindow == nil {
		return
	}

	// 既存のウィンドウを閉じて新しく作成
	st.actionWindow.Close()

	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", world.Resources.UIResources)
	st.actionWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// アクション項目を表示
	for i, action := range st.actionItems {
		isSelected := i == st.actionFocusIndex
		actionWidget := styled.NewListItemText(action, consts.TextColor, isSelected, world.Resources.UIResources)
		windowContainer.AddChild(actionWidget)
	}

	st.actionWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.actionWindow)
}

// closeActionWindow はアクションウィンドウを閉じる
func (st *EquipMenuState) closeActionWindow() {
	if st.actionWindow != nil {
		st.actionWindow.Close()
		st.actionWindow = nil
	}
	st.isWindowMode = false
	st.actionFocusIndex = 0
	st.actionItems = nil
}

// executeActionItem は選択されたアクション項目を実行する
func (st *EquipMenuState) executeActionItem(world w.World) {
	if st.actionFocusIndex >= len(st.actionItems) {
		return
	}

	selectedAction := st.actionItems[st.actionFocusIndex]
	currentTab := st.menuView.GetCurrentTab()
	currentItemIndex := st.menuView.GetCurrentItemIndex()

	if currentItemIndex < 0 || currentItemIndex >= len(currentTab.Items) {
		st.closeActionWindow()
		return
	}

	userData, ok := currentTab.Items[currentItemIndex].UserData.(map[string]interface{})
	if !ok {
		st.closeActionWindow()
		return
	}

	switch selectedAction {
	case "装備する":
		st.startEquipMode(world, userData)
	case "外す":
		st.unequipItem(world, userData)
	case TextClose:
		st.closeActionWindow()
	}
}

// startEquipMode は装備選択モードを開始する
func (st *EquipMenuState) startEquipMode(world w.World, userData map[string]interface{}) {
	member := userData["member"].(ecs.Entity)
	slotNumber := userData["slotNumber"].(gc.EquipmentSlotNumber)
	previousEquipment := userData["entity"].(*ecs.Entity)

	// 現在のタブインデックスを保存
	st.previousTabIndex = st.menuView.GetCurrentTabIndex()

	st.isEquipMode = true
	st.equipSlotNumber = slotNumber
	st.previousEquipment = previousEquipment
	st.equipTargetMember = member // 装備対象のメンバーを保存

	// スロット番号に応じた装備可能なアイテムを取得
	items := st.queryEquipableItemsForSlot(world, slotNumber)
	equipItems := st.createEquipMenuItems(world, items, member)

	newTabs := []tabmenu.TabItem{
		{
			ID:    "equip_selection",
			Label: "装備選択",
			Items: equipItems,
		},
	}

	st.menuView.UpdateTabs(newTabs)
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
	st.closeActionWindow()
}

// createEquipMenuItems は装備選択用のMenuItemを作成する
func (st *EquipMenuState) createEquipMenuItems(world w.World, entities []ecs.Entity, _ ecs.Entity) []tabmenu.Item {
	items := make([]tabmenu.Item, len(entities))

	for i, entity := range entities {
		name := world.Components.Name.Get(entity).(*gc.Name).Name
		items[i] = tabmenu.Item{
			ID:       fmt.Sprintf("equip_entity_%d", entity),
			Label:    name,
			UserData: entity,
		}
	}

	return items
}

// handleEquipItemSelection は装備選択時の処理
func (st *EquipMenuState) handleEquipItemSelection(world w.World, item tabmenu.Item) error {
	entity, ok := item.UserData.(ecs.Entity)
	if !ok {
		return fmt.Errorf("unexpected item UserData")
	}

	// 装備するスロット番号を決定
	slotNumber := st.equipSlotNumber

	// 武器の場合、攻撃タイプに応じてスロット番号を決定
	if entity.HasComponent(world.Components.Weapon) {
		attack := world.Components.Attack.Get(entity).(*gc.Attack)
		if attack != nil {
			switch attack.AttackCategory.Range {
			case gc.AttackRangeMelee:
				slotNumber = gc.SlotMeleeWeapon
			case gc.AttackRangeRanged:
				slotNumber = gc.SlotRangedWeapon
			default:
				return fmt.Errorf("unexpected attack range")
			}
		}
	}

	// 前の装備を外す
	if st.previousEquipment != nil {
		worldhelper.Disarm(world, *st.previousEquipment)
	}

	// 保存されたメンバーに新しい装備を装着
	worldhelper.Equip(world, entity, st.equipTargetMember, slotNumber)

	// 装備モードを終了して元の表示に戻る
	return st.exitEquipMode(world)
}

// unequipItem は装備を外す
func (st *EquipMenuState) unequipItem(world w.World, userData map[string]interface{}) {
	slotEntity, hasEquipment := userData["entity"].(*ecs.Entity)
	if hasEquipment && slotEntity != nil {
		worldhelper.Disarm(world, *slotEntity)
		st.reloadTabs(world)
		st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
		st.reloadAbilityContainer(world)
	}
	st.closeActionWindow()
}

// exitEquipMode は装備選択モードを終了する
func (st *EquipMenuState) exitEquipMode(world w.World) error {
	st.isEquipMode = false
	st.equipSlotNumber = 0
	st.previousEquipment = nil
	st.equipTargetMember = 0 // メンバー情報をクリア

	// 元のタブに戻る
	newTabs := st.createTabs(world)
	st.menuView.UpdateTabs(newTabs)

	// 保存されたタブインデックスに復元
	if st.previousTabIndex >= 0 && st.previousTabIndex < len(newTabs) {
		if err := st.menuView.SetTabIndex(st.previousTabIndex); err != nil {
			return err
		}
	}

	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
	st.reloadAbilityContainer(world)
	return nil
}

// reloadTabs はタブの内容を再読み込みする
func (st *EquipMenuState) reloadTabs(world w.World) {
	newTabs := st.createTabs(world)
	st.menuView.UpdateTabs(newTabs)
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
}
