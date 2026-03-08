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
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// equipSubState は装備メニュー内のサブステート
type equipSubState int

const (
	subStateSlotSelect   equipSubState = iota // スロット選択
	subStateActionWindow                      // アクションウィンドウ
	subStateEquipSelect                       // 装備選択
)

// EquipMenuState は装備メニューのゲームステート
type EquipMenuState struct {
	es.BaseState[w.World]
	subState    equipSubState
	slotMount   *hooks.Mount[slotScreenProps]   // スロット選択画面
	windowMount *hooks.Mount[windowScreenProps] // アクションウィンドウ
	equipMount  *hooks.Mount[equipScreenProps]  // 装備選択画面
	widget      *ebitenui.UI
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
func (st *EquipMenuState) OnStart(_ w.World) error {
	st.subState = subStateSlotSelect
	st.slotMount = hooks.NewMount[slotScreenProps]()
	st.windowMount = hooks.NewMount[windowScreenProps]()
	st.equipMount = hooks.NewMount[equipScreenProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *EquipMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *EquipMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// システム更新
	for _, updater := range []w.Updater{
		&gs.EquipmentChangedSystem{},
		&gs.InventoryChangedSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}

	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.dispatchToCurrentSubState(action)
	}

	// 画面ごとのProps更新
	st.updateSubStateProps(world)

	// dirty判定とUI再構築
	slotDirty := st.slotMount.Update()
	windowDirty := st.windowMount.Update()
	equipDirty := st.equipMount.Update()
	if slotDirty || windowDirty || equipDirty || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *EquipMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// dispatchToCurrentSubState は現在のサブステートにアクションを送る
func (st *EquipMenuState) dispatchToCurrentSubState(action inputmapper.ActionID) {
	switch st.subState {
	case subStateSlotSelect:
		st.slotMount.Dispatch(action)
	case subStateActionWindow:
		st.windowMount.Dispatch(action)
	case subStateEquipSelect:
		st.equipMount.Dispatch(action)
	}
}

// updateSubStateProps は現在のサブステートのPropsを更新する
func (st *EquipMenuState) updateSubStateProps(world w.World) {
	switch st.subState {
	case subStateSlotSelect:
		st.slotMount.SetProps(st.fetchSlotProps(world))
		props := st.slotMount.GetProps()
		itemCounts := make([]int, len(props.Tabs))
		for i, tab := range props.Tabs {
			itemCounts[i] = len(tab.Items)
		}
		hooks.UseTabMenu(st.slotMount.Store(), "slot", hooks.TabMenuConfig{
			TabCount:   len(props.Tabs),
			ItemCounts: itemCounts,
		})
	case subStateActionWindow:
		st.setupWindowState(world)
	case subStateEquipSelect:
		st.equipMount.SetProps(st.fetchEquipProps(world))
		props := st.equipMount.GetProps()
		hooks.UseTabMenu(st.equipMount.Store(), "equip", hooks.TabMenuConfig{
			TabCount:   1,
			ItemCounts: []int{len(props.Items)},
		})
	}
}

// HandleInput はキー入力をActionに変換する
func (st *EquipMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	switch st.subState {
	case subStateSlotSelect, subStateEquipSelect:
		return HandleMenuInput()
	case subStateActionWindow:
		return HandleWindowInput()
	}
	return "", false
}

// DoAction はActionを実行する
func (st *EquipMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch st.subState {
	case subStateActionWindow:
		switch action {
		case inputmapper.ActionWindowConfirm:
			if err := st.executeActionItem(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionWindowCancel:
			st.subState = subStateSlotSelect
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("subStateActionWindow: 未対応のアクション: %s", action)
		}

	case subStateEquipSelect:
		switch action {
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			st.subState = subStateSlotSelect
		case inputmapper.ActionMenuSelect:
			if err := st.handleEquipItemSelection(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("subStateEquipSelect: 未対応のアクション: %s", action)
		}

	case subStateSlotSelect:
		switch action {
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			return es.Transition[w.World]{Type: es.TransPop}, nil
		case inputmapper.ActionMenuSelect:
			if err := st.handleSlotSelection(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("subStateSlotSelect: 未対応のアクション: %s", action)
		}
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

type equipTabData struct {
	ID    string
	Label string
	Items []equipItemData
}

type equipItemData struct {
	SlotLabel  string
	ItemName   string
	SlotNumber gc.EquipmentSlotNumber
	Entity     *ecs.Entity
	Member     ecs.Entity
	// 装備選択モード用
	IsEquipItem bool
	EquipEntity ecs.Entity
}

// slotScreenProps はスロット選択画面のProps
type slotScreenProps struct {
	Tabs             []equipTabData
	Player           ecs.Entity
	PlayerAttributes gc.Attributes
}

// windowScreenProps はアクションウィンドウのProps
type windowScreenProps struct {
	SlotData equipItemData
}

// equipScreenProps は装備選択画面のProps
type equipScreenProps struct {
	Items             []equipItemData
	SlotNumber        gc.EquipmentSlotNumber
	PreviousEquipment *ecs.Entity
	TargetMember      ecs.Entity
}

// fetchSlotProps はスロット選択画面のPropsを取得する
func (st *EquipMenuState) fetchSlotProps(world w.World) slotScreenProps {
	var player ecs.Entity
	var playerFound bool
	worldhelper.QueryPlayer(world, func(entity ecs.Entity) {
		player = entity
		playerFound = true
	})

	tabs := st.createSlotTabs(world, player, playerFound)

	var attrs gc.Attributes
	if playerFound && player.HasComponent(world.Components.Attributes) {
		attrs = *world.Components.Attributes.Get(player).(*gc.Attributes)
	}

	return slotScreenProps{
		Tabs:             tabs,
		Player:           player,
		PlayerAttributes: attrs,
	}
}

// fetchEquipProps は装備選択画面のPropsを取得する
func (st *EquipMenuState) fetchEquipProps(world w.World) equipScreenProps {
	currentProps := st.equipMount.GetProps()
	entities := st.queryEquipableItemsForSlot(world, currentProps.SlotNumber)
	items := make([]equipItemData, len(entities))

	for i, entity := range entities {
		name := world.Components.Name.Get(entity).(*gc.Name).Name
		items[i] = equipItemData{
			ItemName:    name,
			IsEquipItem: true,
			EquipEntity: entity,
		}
	}

	return equipScreenProps{
		Items:             items,
		SlotNumber:        currentProps.SlotNumber,
		PreviousEquipment: currentProps.PreviousEquipment,
		TargetMember:      currentProps.TargetMember,
	}
}

func (st *EquipMenuState) createSlotTabs(world w.World, player ecs.Entity, playerFound bool) []equipTabData {
	// プレイヤーがいない場合でも空のタブを返す
	items := []equipItemData{}
	if playerFound {
		items = st.createAllSlotItems(world, player)
	}
	return []equipTabData{
		{ID: "player_equipment", Label: "装備", Items: items},
	}
}

func (st *EquipMenuState) createAllSlotItems(world w.World, member ecs.Entity) []equipItemData {
	items := make([]equipItemData, 0, 12)

	// 武器スロット
	weapons := worldhelper.GetWeapons(world, member)
	weaponLabels := []string{"武器1", "武器2", "武器3", "武器4", "武器5"}
	weaponSlotNumbers := []gc.EquipmentSlotNumber{
		gc.SlotWeapon1, gc.SlotWeapon2, gc.SlotWeapon3, gc.SlotWeapon4, gc.SlotWeapon5,
	}
	for i, weapon := range weapons {
		itemName := ""
		if weapon != nil {
			itemName = world.Components.Name.Get(*weapon).(*gc.Name).Name
		}
		items = append(items, equipItemData{
			SlotLabel:  weaponLabels[i],
			ItemName:   itemName,
			SlotNumber: weaponSlotNumbers[i],
			Entity:     weapon,
			Member:     member,
		})
	}

	// 防具スロット
	armorSlots := worldhelper.GetArmorEquipments(world, member)
	armorLabels := []string{"防具(頭)", "防具(胴)", "防具(腕)", "防具(手)", "防具(脚)", "防具(足)", "防具(装飾)"}
	armorSlotNumbers := []gc.EquipmentSlotNumber{
		gc.SlotHead, gc.SlotTorso, gc.SlotArms, gc.SlotHands, gc.SlotLegs, gc.SlotFeet, gc.SlotJewelry,
	}
	for i, slot := range armorSlots {
		itemName := ""
		if slot != nil {
			itemName = world.Components.Name.Get(*slot).(*gc.Name).Name
		}
		items = append(items, equipItemData{
			SlotLabel:  armorLabels[i],
			ItemName:   itemName,
			SlotNumber: armorSlotNumbers[i],
			Entity:     slot,
			Member:     member,
		})
	}

	return items
}

func (st *EquipMenuState) queryEquipableItemsForSlot(world w.World, slotNumber gc.EquipmentSlotNumber) []ecs.Entity {
	items := []ecs.Entity{}

	if gc.SlotWeapon1 <= slotNumber && slotNumber <= gc.SlotWeapon5 {
		world.Manager.Join(
			world.Components.Item,
			world.Components.ItemLocationInPlayerBackpack,
			world.Components.Weapon,
		).Visit(ecs.Visit(func(entity ecs.Entity) {
			items = append(items, entity)
		}))
	} else {
		var targetCategory gc.EquipmentType
		switch slotNumber {
		case gc.SlotHead:
			targetCategory = gc.EquipmentHead
		case gc.SlotTorso:
			targetCategory = gc.EquipmentTorso
		case gc.SlotArms:
			targetCategory = gc.EquipmentArms
		case gc.SlotHands:
			targetCategory = gc.EquipmentHands
		case gc.SlotLegs:
			targetCategory = gc.EquipmentLegs
		case gc.SlotFeet:
			targetCategory = gc.EquipmentFeet
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

// ================
// Window
// ================

func (st *EquipMenuState) setupWindowState(world w.World) {
	windowProps := st.windowMount.GetProps()
	actionItems := st.getActionItems(world, windowProps.SlotData)

	hooks.UseState(st.windowMount.Store(), "equip_window_index", 0, func(v int, action inputmapper.ActionID) int {
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

func (st *EquipMenuState) getActionItems(world w.World, item equipItemData) []string {
	if item.SlotLabel == "" {
		return []string{TextClose}
	}

	actionItems := []string{}

	if item.Entity != nil {
		actionItems = append(actionItems, "外す")
	}

	equipableItems := st.queryEquipableItemsForSlot(world, item.SlotNumber)
	if len(equipableItems) > 0 {
		actionItems = append(actionItems, "装備する")
	}

	actionItems = append(actionItems, TextClose)
	return actionItems
}

// handleSlotSelection はスロット選択画面での選択処理
func (st *EquipMenuState) handleSlotSelection(_ w.World) error {
	props := st.slotMount.GetProps()
	tabIndex, ok := hooks.GetState[int](st.slotMount, "slot_tabIndex")
	if !ok {
		return fmt.Errorf("slot_tabIndexの取得に失敗")
	}
	itemIndex, ok := hooks.GetState[int](st.slotMount, "slot_itemIndex")
	if !ok {
		return fmt.Errorf("slot_itemIndexの取得に失敗")
	}

	if tabIndex >= len(props.Tabs) {
		return nil
	}
	tab := props.Tabs[tabIndex]
	if itemIndex >= len(tab.Items) {
		return nil
	}
	item := tab.Items[itemIndex]

	st.subState = subStateActionWindow
	st.windowMount = hooks.NewMount[windowScreenProps]()
	st.windowMount.SetProps(windowScreenProps{
		SlotData: item,
	})
	return nil
}

func (st *EquipMenuState) executeActionItem(world w.World) error {
	windowProps := st.windowMount.GetProps()
	actionIndex, ok := hooks.GetState[int](st.windowMount, "equip_window_index")
	if !ok {
		return fmt.Errorf("equip_window_indexの取得に失敗")
	}
	actionItems := st.getActionItems(world, windowProps.SlotData)

	if actionIndex >= len(actionItems) {
		return nil
	}

	selectedAction := actionItems[actionIndex]
	slotData := windowProps.SlotData

	switch selectedAction {
	case "装備する":
		st.subState = subStateEquipSelect
		st.equipMount = hooks.NewMount[equipScreenProps]()
		st.equipMount.SetProps(equipScreenProps{
			SlotNumber:        slotData.SlotNumber,
			PreviousEquipment: slotData.Entity,
			TargetMember:      slotData.Member,
		})
	case "外す":
		if slotData.Entity != nil {
			worldhelper.MoveToBackpack(world, *slotData.Entity, slotData.Member)
		}
		st.subState = subStateSlotSelect
	case TextClose:
		st.subState = subStateSlotSelect
	}
	return nil
}

func (st *EquipMenuState) handleEquipItemSelection(world w.World) error {
	props := st.equipMount.GetProps()
	itemIndex, ok := hooks.GetState[int](st.equipMount, "equip_itemIndex")
	if !ok {
		return fmt.Errorf("equip_itemIndexの取得に失敗")
	}

	if itemIndex >= len(props.Items) {
		return nil
	}
	item := props.Items[itemIndex]

	// 前の装備を外す
	if props.PreviousEquipment != nil {
		worldhelper.MoveToBackpack(world, *props.PreviousEquipment, props.TargetMember)
	}

	// 新しい装備を装着
	worldhelper.MoveToEquip(world, item.EquipEntity, props.TargetMember, props.SlotNumber)

	st.subState = subStateSlotSelect
	return nil
}

// ================
// buildUI
// ================

func (st *EquipMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources

	root := styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 1行目: タイトル、空、空
	root.AddChild(styled.NewTitleText("装備", res))
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	// サブステートに応じてUIを構築
	switch st.subState {
	case subStateEquipSelect:
		props := st.equipMount.GetProps()
		itemIndex, _ := hooks.GetState[int](st.equipMount, "equip_itemIndex")
		root.AddChild(st.buildEquipSelectContainer(props, itemIndex, res))
		root.AddChild(widget.NewContainer())
		root.AddChild(st.buildEquipDetailContainer(world, props, itemIndex, res))
		root.AddChild(st.buildEquipDescContainer(world, props.Items, itemIndex, res))
	default: // screenSlotSelect, screenActionWindow
		slotProps := st.slotMount.GetProps()
		tabIndex, _ := hooks.GetState[int](st.slotMount, "slot_tabIndex")
		itemIndex, _ := hooks.GetState[int](st.slotMount, "slot_itemIndex")
		root.AddChild(st.buildSlotContainer(slotProps.Tabs, tabIndex, itemIndex, res))
		root.AddChild(widget.NewContainer())
		root.AddChild(st.buildSlotDetailContainer(world, slotProps, tabIndex, itemIndex, res))
		root.AddChild(st.buildSlotDescContainer(world, slotProps.Tabs, tabIndex, itemIndex, res))
	}
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	eui := &ebitenui.UI{Container: root}

	// アクションウィンドウを追加
	if st.subState == subStateActionWindow {
		windowProps := st.windowMount.GetProps()
		actionWindow := st.buildActionWindow(world, windowProps)
		eui.AddWindow(actionWindow)
	}

	return eui
}

// buildSlotContainer はスロット選択画面のアイテム一覧を構築する
func (st *EquipMenuState) buildSlotContainer(tabs []equipTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(tabs) {
		return container
	}

	currentTab := tabs[tabIndex]
	columnWidths := []int{20, 80, 120}
	table := styled.NewTableContainer(columnWidths, res)
	for i, item := range currentTab.Items {
		isSelected := i == itemIndex
		styled.NewTableRow(table, columnWidths, []string{"", item.SlotLabel, item.ItemName}, nil, &isSelected, res)
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(装備なし)", res))
	}

	return container
}

// buildEquipSelectContainer は装備選択画面のアイテム一覧を構築する
func (st *EquipMenuState) buildEquipSelectContainer(props equipScreenProps, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()

	columnWidths := []int{20, 150}
	table := styled.NewTableContainer(columnWidths, res)
	for i, item := range props.Items {
		isSelected := i == itemIndex
		styled.NewTableRow(table, columnWidths, []string{"", item.ItemName}, nil, &isSelected, res)
	}
	container.AddChild(table)

	if len(props.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(装備なし)", res))
	}

	return container
}

// buildSlotDetailContainer はスロット選択画面の詳細表示を構築する
func (st *EquipMenuState) buildSlotDetailContainer(world w.World, props slotScreenProps, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	specContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	abilityContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 性能表示
	if tabIndex < len(props.Tabs) && itemIndex < len(props.Tabs[tabIndex].Items) {
		item := props.Tabs[tabIndex].Items[itemIndex]
		if item.Entity != nil {
			views.UpdateSpec(world, specContainer, *item.Entity)
		}
	}

	// 能力表示
	st.buildAbilityDisplay(world, abilityContainer, props.Player, res)

	return styled.NewWSplitContainer(specContainer, abilityContainer)
}

// buildEquipDetailContainer は装備選択画面の詳細表示を構築する
func (st *EquipMenuState) buildEquipDetailContainer(world w.World, props equipScreenProps, itemIndex int, res *resources.UIResources) *widget.Container {
	specContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	abilityContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 性能表示
	if itemIndex < len(props.Items) {
		item := props.Items[itemIndex]
		views.UpdateSpec(world, specContainer, item.EquipEntity)
	}

	// 能力表示(装備選択中もプレイヤー能力を表示)
	slotProps := st.slotMount.GetProps()
	st.buildAbilityDisplay(world, abilityContainer, slotProps.Player, res)

	return styled.NewWSplitContainer(specContainer, abilityContainer)
}

func (st *EquipMenuState) buildAbilityDisplay(world w.World, container *widget.Container, player ecs.Entity, res *resources.UIResources) {
	if !player.HasComponent(world.Components.Player) {
		return
	}

	views.AddMemberStatusText(container, player, world)

	if !player.HasComponent(world.Components.Attributes) {
		return
	}

	columnWidths := []int{50, 30, 40}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight, styled.AlignRight}

	attrs := world.Components.Attributes.Get(player).(*gc.Attributes)

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableRow(table, columnWidths, []string{consts.VitalityLabel, fmt.Sprintf("%d", attrs.Vitality.Total), fmt.Sprintf("(%+d)", attrs.Vitality.Modifier)}, aligns, nil, res)
	styled.NewTableRow(table, columnWidths, []string{consts.StrengthLabel, fmt.Sprintf("%d", attrs.Strength.Total), fmt.Sprintf("(%+d)", attrs.Strength.Modifier)}, aligns, nil, res)
	styled.NewTableRow(table, columnWidths, []string{consts.SensationLabel, fmt.Sprintf("%d", attrs.Sensation.Total), fmt.Sprintf("(%+d)", attrs.Sensation.Modifier)}, aligns, nil, res)
	styled.NewTableRow(table, columnWidths, []string{consts.DexterityLabel, fmt.Sprintf("%d", attrs.Dexterity.Total), fmt.Sprintf("(%+d)", attrs.Dexterity.Modifier)}, aligns, nil, res)
	styled.NewTableRow(table, columnWidths, []string{consts.AgilityLabel, fmt.Sprintf("%d", attrs.Agility.Total), fmt.Sprintf("(%+d)", attrs.Agility.Modifier)}, aligns, nil, res)
	styled.NewTableRow(table, columnWidths, []string{consts.DefenseLabel, fmt.Sprintf("%d", attrs.Defense.Total), fmt.Sprintf("(%+d)", attrs.Defense.Modifier)}, aligns, nil, res)
	container.AddChild(table)
}

// buildSlotDescContainer はスロット選択画面の説明文を構築する
func (st *EquipMenuState) buildSlotDescContainer(world w.World, tabs []equipTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	desc := " "

	if tabIndex < len(tabs) && itemIndex < len(tabs[tabIndex].Items) {
		item := tabs[tabIndex].Items[itemIndex]
		if item.Entity != nil && (*item.Entity).HasComponent(world.Components.Description) {
			descComp := world.Components.Description.Get(*item.Entity).(*gc.Description)
			desc = descComp.Description
		}
	}

	if desc == "" {
		desc = " "
	}
	container.AddChild(styled.NewMenuText(desc, res))
	return container
}

// buildEquipDescContainer は装備選択画面の説明文を構築する
func (st *EquipMenuState) buildEquipDescContainer(world w.World, items []equipItemData, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	desc := " "

	if itemIndex < len(items) {
		item := items[itemIndex]
		if item.EquipEntity.HasComponent(world.Components.Description) {
			descComp := world.Components.Description.Get(item.EquipEntity).(*gc.Description)
			desc = descComp.Description
		}
	}

	if desc == "" {
		desc = " "
	}
	container.AddChild(styled.NewMenuText(desc, res))
	return container
}

func (st *EquipMenuState) buildActionWindow(world w.World, windowProps windowScreenProps) *widget.Window {
	res := world.Resources.UIResources
	actionIndex, _ := hooks.GetState[int](st.windowMount, "equip_window_index")
	actionItems := st.getActionItems(world, windowProps.SlotData)

	windowContainer := styled.NewWindowContainer(res)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", res)
	window := styled.NewSmallWindow(titleContainer, windowContainer)

	for i, action := range actionItems {
		isSelected := i == actionIndex
		actionWidget := styled.NewListItemText(action, consts.TextColor, isSelected, res)
		windowContainer.AddChild(actionWidget)
	}

	window.SetLocation(getCenterWinRect(world))
	return window
}
