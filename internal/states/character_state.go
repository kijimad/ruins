package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 画面タブメニュー。キャラクター情報の閲覧・操作を装備・スキルのタブでまとめる。
// 所持アイテムへの動詞でなくキャラクター情報が対象なので、動詞タブ画面とは別 state にする。
// ダンジョンからの直達ショートカットは持たず、ダンジョンメニューから開く。

const characterMenuKey = "character"

// characterSub は画面タブメニュー内のサブステート
type characterSub int

const (
	charSubBrowse       characterSub = iota // タブとカーソルの操作
	charSubActionWindow                     // 装備スロットのアクション選択
	charSubEquipSelect                      // 装備するアイテムの選択
)

// 画面タブのインデックス
const (
	charScreenEquip  = 0
	charScreenSkills = 1
)

// CharacterState は画面タブメニューのステート
type CharacterState struct {
	es.BaseState[w.World]
	subState    characterSub
	showDetail  bool // x の詳細モーダルを表示中か
	rebuild     bool // 次フレームで UI を作り直すか
	mount       *hooks.Mount[characterProps]
	windowMount *hooks.Mount[charWindowProps]
	equipMount  *hooks.Mount[charEquipProps]
	widget      *ebitenui.UI
}

var _ es.State[w.World] = &CharacterState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CharacterState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CharacterState) OnResume(_ w.World) error { return nil }

// OnStop はステートが終了する際に呼ばれる
func (st *CharacterState) OnStop(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *CharacterState) OnStart(_ w.World) error {
	st.subState = charSubBrowse
	st.mount = hooks.NewMount[characterProps]()
	st.windowMount = hooks.NewMount[charWindowProps]()
	st.equipMount = hooks.NewMount[charEquipProps]()
	return nil
}

// Update はステートの更新処理
func (st *CharacterState) Update(world w.World) (es.Transition[w.World], error) {
	for _, updater := range []w.Updater{
		&gs.StatsChangedSystem{},
		&gs.WeightDirtySystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}

	if action, ok := st.handleInput(); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.dispatch(action)
	}

	st.mount.SetProps(st.fetchProps(world))
	props := st.mount.GetProps()

	// 画面タブのカーソル。左右キーはタブ切替に読み替える
	itemCounts := []int{len(props.EquipSlots), len(props.Skills)}
	skills := props.Skills
	skillSkips := make([]bool, len(skills))
	for i, s := range skills {
		skillSkips[i] = s.IsHeader
	}
	hooks.UseTabMenu(st.mount.Store(), characterMenuKey, hooks.TabMenuConfig{
		TabCount:   2,
		ItemCounts: itemCounts,
		Skips:      [][]bool{make([]bool, len(props.EquipSlots)), skillSkips},
	})

	// 装備選択サブステートのカーソル
	if st.subState == charSubEquipSelect {
		eprops := st.equipMount.GetProps()
		hooks.UseTabMenu(st.equipMount.Store(), "char_equip", hooks.TabMenuConfig{
			TabCount:   1,
			ItemCounts: []int{len(eprops.Items)},
		})
	}
	// アクションウィンドウのカーソル
	if st.subState == charSubActionWindow {
		st.setupWindowState(world)
	}

	menuDirty := st.mount.Update()
	windowDirty := st.windowMount.Update()
	equipDirty := st.equipMount.Update()
	if menuDirty || windowDirty || equipDirty || st.widget == nil || st.rebuild {
		st.widget = st.buildUI(world)
		st.rebuild = false
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理
func (st *CharacterState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// dispatch は現在のサブステートにアクションを送る。閲覧中の左右キーはタブ切替へ読み替える
func (st *CharacterState) dispatch(action inputmapper.ActionID) {
	switch st.subState {
	case charSubBrowse:
		d := action
		switch action {
		case inputmapper.ActionMenuLeft:
			d = inputmapper.ActionMenuTabPrev
		case inputmapper.ActionMenuRight:
			d = inputmapper.ActionMenuTabNext
		default:
		}
		st.mount.Dispatch(d)
	case charSubEquipSelect:
		st.equipMount.Dispatch(action)
	case charSubActionWindow:
		st.windowMount.Dispatch(action)
	}
}

// handleInput はキー入力を Action に変換する
func (st *CharacterState) handleInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if st.showDetail {
		if ki.IsKeyJustPressed(ebiten.KeyEscape) || ki.IsKeyJustPressed(ebiten.KeyX) || ki.IsEnterJustPressedOnce() {
			return inputmapper.ActionMenuCancel, true
		}
		return "", false
	}
	switch st.subState {
	case charSubActionWindow:
		return HandleWindowInput()
	case charSubBrowse, charSubEquipSelect:
		if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
			return inputmapper.ActionOpenItemDetail, true
		}
		return HandleMenuInput()
	}
	return "", false
}

// DoAction は Action を実行する
func (st *CharacterState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	if st.showDetail {
		if action == inputmapper.ActionMenuCancel {
			st.showDetail = false
			st.rebuild = true
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	switch st.subState {
	case charSubActionWindow:
		return st.doActionWindow(world, action)
	case charSubEquipSelect:
		return st.doEquipSelect(world, action)
	case charSubBrowse:
		return st.doBrowse(action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

func (st *CharacterState) doBrowse(action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.showDetail = true
		st.rebuild = true
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSelect:
		st.onBrowseSelect()
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// onBrowseSelect は閲覧中の Enter を処理する。装備タブはアクションウィンドウ、スキルタブは詳細モーダルを開く
func (st *CharacterState) onBrowseSelect() {
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, characterMenuKey)
	if menuState.TabIndex == charScreenSkills {
		st.showDetail = true
		st.rebuild = true
		return
	}
	props := st.mount.GetProps()
	if menuState.ItemIndex >= len(props.EquipSlots) {
		return
	}
	st.windowMount = hooks.NewMount[charWindowProps]()
	st.windowMount.SetProps(charWindowProps{SlotData: props.EquipSlots[menuState.ItemIndex]})
	st.subState = charSubActionWindow
	st.rebuild = true
}

func (st *CharacterState) doActionWindow(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionWindowConfirm:
		if err := st.executeSlotAction(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		st.rebuild = true
	case inputmapper.ActionWindowCancel:
		st.subState = charSubBrowse
		st.rebuild = true
	case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
		// Dispatch で処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("アクションウィンドウ: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

func (st *CharacterState) doEquipSelect(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		st.subState = charSubBrowse
		st.rebuild = true
	case inputmapper.ActionMenuSelect:
		if err := st.executeEquip(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		st.rebuild = true
	case inputmapper.ActionOpenItemDetail:
		st.showDetail = true
		st.rebuild = true
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight:
		// Dispatch で処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("装備選択: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// setupWindowState はアクションウィンドウのカーソルを登録する
func (st *CharacterState) setupWindowState(world w.World) {
	windowProps := st.windowMount.GetProps()
	actionCount := len(st.slotActions(world, windowProps.SlotData))
	hooks.UseState(st.windowMount.Store(), "char_window_index", 0, func(v int, a inputmapper.ActionID) int {
		if actionCount == 0 {
			return 0
		}
		switch a {
		case inputmapper.ActionWindowUp:
			return (v - 1 + actionCount) % actionCount
		case inputmapper.ActionWindowDown:
			return (v + 1) % actionCount
		default:
			return v
		}
	})
}

// slotActions はスロットに対して選べるアクションを返す
func (st *CharacterState) slotActions(world w.World, slot equipItemData) []string {
	actions := []string{}
	if slot.Entity != nil {
		actions = append(actions, "外す")
	}
	if len(equipableForSlot(world, slot.SlotNumber)) > 0 {
		actions = append(actions, TextEquip)
	}
	actions = append(actions, TextClose)
	return actions
}

// executeSlotAction はアクションウィンドウで選んだアクションを実行する
func (st *CharacterState) executeSlotAction(world w.World) error {
	windowProps := st.windowMount.GetProps()
	idx, _ := hooks.GetState[int](st.windowMount, "char_window_index")
	actions := st.slotActions(world, windowProps.SlotData)
	if idx >= len(actions) {
		return nil
	}
	slot := windowProps.SlotData
	switch actions[idx] {
	case TextEquip:
		st.equipMount = hooks.NewMount[charEquipProps]()
		st.equipMount.SetProps(charEquipProps{
			SlotNumber:        slot.SlotNumber,
			PreviousEquipment: slot.Entity,
			TargetMember:      slot.Member,
		})
		st.subState = charSubEquipSelect
	case "外す":
		if slot.Entity != nil {
			if err := lifecycle.MoveToBackpack(world, *slot.Entity, slot.Member); err != nil {
				return err
			}
		}
		st.subState = charSubBrowse
	case TextClose:
		st.subState = charSubBrowse
	}
	return nil
}

// executeEquip は装備選択で選んだアイテムを装着する
func (st *CharacterState) executeEquip(world w.World) error {
	props := st.equipMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.equipMount, "char_equip")
	if menuState.ItemIndex >= len(props.Items) {
		return nil
	}
	item := props.Items[menuState.ItemIndex]

	if props.PreviousEquipment != nil {
		if err := lifecycle.MoveToBackpack(world, *props.PreviousEquipment, props.TargetMember); err != nil {
			return err
		}
	}
	lifecycle.MoveToEquip(world, item, props.TargetMember, props.SlotNumber)
	st.subState = charSubBrowse
	return nil
}

// ================
// Props
// ================

type characterProps struct {
	EquipSlots []equipItemData
	Skills     []characterSkillRow
}

type characterSkillRow struct {
	Label    string
	Value    string
	Desc     string
	IsHeader bool
}

// charWindowProps はアクションウィンドウの Props
type charWindowProps struct {
	SlotData equipItemData
}

// charEquipProps は装備選択の Props
type charEquipProps struct {
	Items             []ecs.Entity
	SlotNumber        gc.EquipmentSlotNumber
	PreviousEquipment *ecs.Entity
	TargetMember      ecs.Entity
}

func (st *CharacterState) fetchProps(world w.World) characterProps {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return characterProps{}
	}
	return characterProps{
		EquipSlots: playerEquipSlots(world, player),
		Skills:     playerSkillRows(world, player),
	}
}

// playerEquipSlots はプレイヤーの全装備スロットを列挙する
func playerEquipSlots(world w.World, player ecs.Entity) []equipItemData {
	items := make([]equipItemData, 0, 12)

	weapons := query.GetWeapons(world, player)
	weaponLabels := []string{"武器1", "武器2", "武器3", "武器4", "武器5"}
	weaponSlots := []gc.EquipmentSlotNumber{gc.SlotWeapon1, gc.SlotWeapon2, gc.SlotWeapon3, gc.SlotWeapon4, gc.SlotWeapon5}
	for i, weapon := range weapons {
		name := ""
		if weapon != nil {
			name = world.Components.Name.Get(*weapon).Name
		}
		items = append(items, equipItemData{SlotLabel: weaponLabels[i], ItemName: name, SlotNumber: weaponSlots[i], Entity: weapon, Member: player})
	}

	armor := query.GetArmorEquipments(world, player)
	armorLabels := []string{"防具(頭)", "防具(胴)", "防具(腕)", "防具(手)", "防具(脚)", "防具(足)", "防具(装飾)"}
	armorSlots := []gc.EquipmentSlotNumber{gc.SlotHead, gc.SlotTorso, gc.SlotArms, gc.SlotHands, gc.SlotLegs, gc.SlotFeet, gc.SlotJewelry}
	for i, slot := range armor {
		name := ""
		if slot != nil {
			name = world.Components.Name.Get(*slot).Name
		}
		items = append(items, equipItemData{SlotLabel: armorLabels[i], ItemName: name, SlotNumber: armorSlots[i], Entity: slot, Member: player})
	}

	return items
}

// playerSkillRows はプレイヤーのスキル一覧をカテゴリ見出し付きで返す
func playerSkillRows(world w.World, player ecs.Entity) []characterSkillRow {
	rows := []characterSkillRow{}
	if !aliveHas(world, world.Components.Skills, player) {
		return rows
	}
	skills := world.Components.Skills.Get(player)
	for _, cat := range gc.SkillCategories {
		rows = append(rows, characterSkillRow{Label: cat.Name, IsHeader: true, Desc: fmt.Sprintf("%sカテゴリのスキル", cat.Name)})
		for _, id := range cat.IDs {
			s := skills.Get(id)
			expFrac := 0
			if s.Exp.Max > 0 {
				expFrac = s.Exp.Current * 1000 / s.Exp.Max
			}
			info := gc.SkillDescription(id)
			rows = append(rows, characterSkillRow{
				Label: gc.SkillName(id),
				Value: fmt.Sprintf("%d.%03d", s.Value, expFrac),
				Desc:  info.Summary,
			})
		}
	}
	return rows
}

// equipableForSlot は指定スロットに装備できるバックパック内アイテムを返す
func equipableForSlot(world w.World, slotNumber gc.EquipmentSlotNumber) []ecs.Entity {
	items := []ecs.Entity{}
	if gc.SlotWeapon1 <= slotNumber && slotNumber <= gc.SlotWeapon5 {
		q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
		for q.Next() {
			entity := q.Entity()
			cat, _ := world.Components.CategoryOf(gc.InventoryCategoryKey, entity)
			if cat != gc.CategoryWeapon {
				continue
			}
			items = append(items, entity)
		}
		return query.SortEntities(world, items)
	}

	var target gc.EquipmentType
	switch slotNumber {
	case gc.SlotHead:
		target = gc.EquipmentHead
	case gc.SlotTorso:
		target = gc.EquipmentTorso
	case gc.SlotArms:
		target = gc.EquipmentArms
	case gc.SlotHands:
		target = gc.EquipmentHands
	case gc.SlotLegs:
		target = gc.EquipmentLegs
	case gc.SlotFeet:
		target = gc.EquipmentFeet
	case gc.SlotJewelry:
		target = gc.EquipmentJewelry
	default:
		return query.SortEntities(world, items)
	}

	q := ecs.NewFilter2[gc.LocationInBackpack, gc.Wearable](world.ECS).Query()
	for q.Next() {
		entity := q.Entity()
		wearable := world.Components.Wearable.Get(entity)
		if wearable != nil && wearable.EquipmentCategory == target {
			items = append(items, entity)
		}
	}
	return query.SortEntities(world, items)
}

// ================
// buildUI
// ================

func (st *CharacterState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, characterMenuKey)
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	root := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(1),
				widget.GridLayoutOpts.Spacing(0, theme.Space2),
				widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, false, true, false}),
				widget.GridLayoutOpts.Padding(&widget.Insets{Top: theme.Space3, Bottom: theme.Space3, Left: theme.Space3, Right: theme.Space3}),
			),
		),
	)

	root.AddChild(styled.NewTitleText("キャラクター", res))

	tabRow := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	tabBar := styled.NewTabBar([]string{"装備", "スキル"}, tabIndex, res)
	tabBar.GetWidget().LayoutData = widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter}
	tabRow.AddChild(tabBar)
	root.AddChild(tabRow)

	if tabIndex == charScreenEquip {
		root.AddChild(st.buildEquipList(props.EquipSlots, itemIndex, res))
		root.AddChild(st.buildEquipDesc(props.EquipSlots, itemIndex, res))
	} else {
		root.AddChild(st.buildSkillList(props.Skills, itemIndex, res))
		root.AddChild(st.buildSkillDesc(props.Skills, itemIndex, res))
	}

	ui := &ebitenui.UI{Container: root}

	if st.subState == charSubActionWindow {
		ui.AddWindow(st.buildActionWindow(world, res))
	}
	if st.subState == charSubEquipSelect {
		ui.AddWindow(st.buildEquipSelectWindow(world, res))
	}
	if st.showDetail {
		if win := st.buildDetailWindow(world, props, tabIndex, itemIndex, res); win != nil {
			ui.AddWindow(win)
		}
	}

	return ui
}

func (st *CharacterState) buildEquipList(slots []equipItemData, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if len(slots) == 0 {
		container.AddChild(styled.NewDescriptionText("装備スロットがありません", res))
		return container
	}
	for i, slot := range slots {
		isSelected := i == itemIndex
		clr := theme.TextSecondary
		if isSelected {
			clr = theme.TextPrimary
		}
		name := slot.ItemName
		if name == "" {
			name = "（なし）"
		}
		container.AddChild(styled.NewListItemText(fmt.Sprintf("%s  %s", slot.SlotLabel, name), clr, isSelected, res))
	}
	return container
}

func (st *CharacterState) buildSkillList(rows []characterSkillRow, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if len(rows) == 0 {
		container.AddChild(styled.NewDescriptionText("スキルがありません", res))
		return container
	}
	for i, row := range rows {
		if row.IsHeader {
			container.AddChild(styled.NewDescriptionText(row.Label, res))
			continue
		}
		isSelected := i == itemIndex
		clr := theme.TextSecondary
		if isSelected {
			clr = theme.TextPrimary
		}
		container.AddChild(styled.NewListItemText(fmt.Sprintf("%s  %s", row.Label, row.Value), clr, isSelected, res))
	}
	return container
}

func (st *CharacterState) buildEquipDesc(slots []equipItemData, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	text := " "
	if itemIndex < len(slots) {
		name := slots[itemIndex].ItemName
		if name == "" {
			text = fmt.Sprintf("%s は空き   Enter で装備", slots[itemIndex].SlotLabel)
		} else {
			text = fmt.Sprintf("%s を選択中   Enter で着脱   x で詳細", name)
		}
	}
	container.AddChild(styled.NewMenuText(text, res))
	return container
}

func (st *CharacterState) buildSkillDesc(rows []characterSkillRow, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	text := " "
	if itemIndex < len(rows) {
		text = rows[itemIndex].Desc
	}
	if text == "" {
		text = " "
	}
	container.AddChild(styled.NewMenuText(text, res))
	return container
}

func (st *CharacterState) buildActionWindow(world w.World, res resources.UIResources) *widget.Window {
	windowProps := st.windowMount.GetProps()
	idx, _ := hooks.GetState[int](st.windowMount, "char_window_index")
	actions := st.slotActions(world, windowProps.SlotData)

	content := styled.NewWindowContainer(res)
	title := styled.NewWindowHeaderContainer("アクション選択", res)
	win := styled.NewSmallWindow(title, content)
	for i, action := range actions {
		content.AddChild(styled.NewListItemText(action, theme.TextSecondary, i == idx, res))
	}
	win.SetLocation(getCenterWinRect(world))
	return win
}

func (st *CharacterState) buildEquipSelectWindow(world w.World, res resources.UIResources) *widget.Window {
	props := st.equipMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.equipMount, "char_equip")

	content := styled.NewWindowContainer(res)
	title := styled.NewWindowHeaderContainer("装備を選ぶ", res)
	win := styled.NewSmallWindow(title, content)
	if len(props.Items) == 0 {
		content.AddChild(styled.NewDescriptionText("装備できるものがない", res))
	}
	for i, entity := range props.Items {
		name := world.Components.Name.Get(entity).Name
		content.AddChild(styled.NewListItemText(name, theme.TextSecondary, i == menuState.ItemIndex, res))
	}
	win.SetLocation(getCenterWinRect(world))
	return win
}

func (st *CharacterState) buildDetailWindow(world w.World, props characterProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Window {
	var titleText, body string
	if tabIndex == charScreenSkills {
		if itemIndex >= len(props.Skills) {
			return nil
		}
		row := props.Skills[itemIndex]
		titleText = row.Label
		body = row.Desc
	} else {
		if itemIndex >= len(props.EquipSlots) {
			return nil
		}
		slot := props.EquipSlots[itemIndex]
		if slot.Entity == nil {
			titleText = slot.SlotLabel
			body = "何も装備していない"
		} else {
			titleText = slot.ItemName
			if world.Components.Description.Has(*slot.Entity) {
				body = world.Components.Description.Get(*slot.Entity).Description
			}
		}
	}
	if body == "" {
		body = "説明はない"
	}

	content := styled.NewWindowContainer(res)
	content.AddChild(styled.NewDescriptionText(body, res))
	title := styled.NewWindowHeaderContainer(titleText, res)
	win := styled.NewSmallWindow(title, content)
	win.SetLocation(getCenterWinRect(world))
	return win
}
