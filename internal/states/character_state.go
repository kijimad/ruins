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
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/views"
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

// 画面タブ。装備は編集可能、以降は読み取り専用の情報タブ
const charScreenEquip = 0

// characterTabLabels は画面タブの見出し。装備の後ろに読み取り専用タブが並ぶ
var characterTabLabels = []string{"装備", "能力", "スキル", "効果", "健康", "基本"}

// CharacterState は画面タブメニューのステート。主人公と仲間で同じ画面を使い、対象を切り替えられる
type CharacterState struct {
	es.BaseState[w.World]
	target      ecs.Entity // 表示対象のキャラクター。ゼロ値なら主人公
	subState    characterSub
	showDetail  bool // x の詳細モーダルを表示中か
	rebuild     bool // 次フレームで UI を作り直すか
	mount       *hooks.Mount[characterProps]
	windowMount *hooks.Mount[charWindowProps]
	equipMount  *hooks.Mount[charEquipProps]
	widget      *ebitenui.UI
}

var _ es.State[w.World] = &CharacterState{}
var _ Configurable = &CharacterState{}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *CharacterState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

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

	// 対象キャラの切り替え。閲覧中のみ [ ] で主人公と仲間を巡回する
	if st.subState == charSubBrowse && !st.showDetail {
		ki := input.GetSharedKeyboardInput()
		if ki.IsKeyJustPressed(ebiten.KeyBracketRight) {
			st.switchMember(world, 1)
		} else if ki.IsKeyJustPressed(ebiten.KeyBracketLeft) {
			st.switchMember(world, -1)
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

	// 画面タブのカーソル。装備 + 情報タブ。左右キーはタブ切替に読み替える
	itemCounts := make([]int, 0, 1+len(props.InfoTabs))
	skips := make([][]bool, 0, 1+len(props.InfoTabs))
	itemCounts = append(itemCounts, len(props.EquipSlots))
	skips = append(skips, make([]bool, len(props.EquipSlots)))
	for _, tab := range props.InfoTabs {
		itemCounts = append(itemCounts, len(tab.Items))
		s := make([]bool, len(tab.Items))
		for i, item := range tab.Items {
			s[i] = item.IsHeader
		}
		skips = append(skips, s)
	}
	hooks.UseTabMenu(st.mount.Store(), characterMenuKey, hooks.TabMenuConfig{
		TabCount:   len(itemCounts),
		ItemCounts: itemCounts,
		Skips:      skips,
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

// onBrowseSelect は閲覧中の Enter を処理する。装備タブはアクションウィンドウ、情報タブは詳細モーダルを開く
func (st *CharacterState) onBrowseSelect() {
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, characterMenuKey)
	if menuState.TabIndex != charScreenEquip {
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
	TargetName  string // 表示対象のキャラクター名
	HasMultiple bool   // 切り替え可能な仲間がいるか
	EquipSlots  []equipItemData
	InfoTabs    []statusTabData // 能力・スキル・効果・健康・基本の読み取り専用タブ
}

// equipItemData は装備スロット1つ分の表示データ
type equipItemData struct {
	SlotLabel  string
	ItemName   string
	SlotNumber gc.EquipmentSlotNumber
	Entity     *ecs.Entity // 装備中エンティティ。空きなら nil
	Member     ecs.Entity
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
	target := st.resolveTarget(world)
	if !world.ECS.Alive(target) {
		return characterProps{}
	}
	name := ""
	if world.Components.Name.Has(target) {
		name = query.GetEntityName(target, world)
	}
	return characterProps{
		TargetName:  name,
		HasMultiple: len(characterMembers(world)) > 1,
		EquipSlots:  memberEquipSlots(world, target),
		InfoTabs:    st.fetchInfoTabs(world, target),
	}
}

// resolveTarget は表示対象を返す。未指定または死亡時は主人公にフォールバックする
func (st *CharacterState) resolveTarget(world w.World) ecs.Entity {
	if world.ECS.Alive(st.target) {
		return st.target
	}
	if player, err := query.GetPlayerEntity(world); err == nil {
		return player
	}
	return st.target
}

// characterMembers は切り替え対象、主人公と仲間、を表示順に返す
func characterMembers(world w.World) []ecs.Entity {
	members := []ecs.Entity{}
	if player, err := query.GetPlayerEntity(world); err == nil {
		members = append(members, player)
	}
	members = append(members, query.SquadMembers(world)...)
	return members
}

// switchMember は表示対象を dir 方向の隣のキャラへ巡回で切り替える
func (st *CharacterState) switchMember(world w.World, dir int) {
	members := characterMembers(world)
	if len(members) <= 1 {
		return
	}
	cur := st.resolveTarget(world)
	idx := 0
	for i, m := range members {
		if m == cur {
			idx = i
			break
		}
	}
	st.target = members[(idx+dir+len(members))%len(members)]
	st.rebuild = true
}

// memberEquipSlots はプレイヤーの全装備スロットを列挙する
func memberEquipSlots(world w.World, player ecs.Entity) []equipItemData {
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
				// 名前・タブ・コンテンツを上詰めし、スペーサー行を伸ばしてフッターを下端へ押す。
				// これでコンテンツの開始位置がタブ内容量によらず一定になる
				widget.GridLayoutOpts.Stretch([]bool{true}, []bool{false, false, false, true, false}),
				widget.GridLayoutOpts.Padding(&widget.Insets{Top: theme.Space3, Bottom: theme.Space3, Left: theme.Space3, Right: theme.Space3}),
			),
		),
	)

	// Row 0: 対象キャラ名。仲間がいれば左右矢印で切替可能を示す。汎用タイトルは置かない。
	// 矢印は素の記号だとフォントに無く文字化けするため FontAwesome のアイコンを使う
	nameText := props.TargetName
	if props.HasMultiple {
		nameText = fmt.Sprintf("%s %s %s", consts.IconArrowLeft, props.TargetName, consts.IconArrowRight)
	}
	nameRow := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	nameLabel := styled.NewMenuText(nameText, res)
	nameLabel.GetWidget().LayoutData = widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter}
	nameRow.AddChild(nameLabel)
	root.AddChild(nameRow)

	// Row 1: タブ帯
	tabRow := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	tabBar := styled.NewTabBar(characterTabLabels, tabIndex, res)
	tabBar.GetWidget().LayoutData = widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter}
	tabRow.AddChild(tabBar)
	root.AddChild(tabRow)

	// Row 2: コンテンツ。上詰めで置き、タブによらず同じ位置から始める。説明の常時表示は置かない
	if tabIndex == charScreenEquip {
		root.AddChild(st.buildEquipList(props.EquipSlots, itemIndex, res))
	} else if infoIdx := tabIndex - 1; infoIdx < len(props.InfoTabs) {
		root.AddChild(st.buildInfoTable(props.InfoTabs[infoIdx], itemIndex, res))
	} else {
		root.AddChild(widget.NewContainer())
	}

	// Row 3: 伸縮スペーサー。フッターを下端へ押す
	root.AddChild(widget.NewContainer())

	// Row 4: キー案内。切替は仲間がいるときだけ出す
	hint := "x で詳細"
	if props.HasMultiple {
		hint = "[ ] で切替   x で詳細"
	}
	hintRow := styled.NewRowContainer()
	hintRow.AddChild(styled.NewDescriptionText(hint, res))
	root.AddChild(hintRow)

	// 後ろのフィールドを見せるため、モーダルを画面より一回り小さい中央ボックスにする
	outer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true}),
			widget.GridLayoutOpts.Padding(&widget.Insets{Top: 48, Bottom: 48, Left: 96, Right: 96}),
		)),
	)
	outer.AddChild(root)

	ui := &ebitenui.UI{Container: outer}

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
	// 情報タブと開始位置を揃えるため、先頭に同じページ表示行を必ず置く
	container.AddChild(newPageIndicatorRow(itemIndex, len(slots), res))
	if len(slots) == 0 {
		container.AddChild(styled.NewDescriptionText("装備スロットがありません", res))
		return container
	}

	// 情報タブと同じテーブル描画に揃える。左にスロット名、右に装備名。未装備は空欄
	columnWidths := []int{110, 220}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft}
	table := styled.NewTableContainer(columnWidths, res)
	for i, slot := range slots {
		isSelected := i == itemIndex
		styled.NewTableRow(table, columnWidths, []string{slot.SlotLabel, slot.ItemName}, aligns, &isSelected, res)
	}
	container.AddChild(table)
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
	content := styled.NewWindowContainer(res)

	if tabIndex == charScreenEquip {
		if itemIndex >= len(props.EquipSlots) {
			return nil
		}
		slot := props.EquipSlots[itemIndex]
		if slot.Entity == nil {
			content.AddChild(styled.NewMenuText(slot.SlotLabel, res))
			content.AddChild(styled.NewDescriptionText("何も装備していない", res))
		} else {
			// 装備の性能・性質を細かく出す
			content.AddChild(styled.NewMenuText(slot.ItemName, res))
			spec := styled.NewVerticalContainer()
			views.UpdateSpec(world, spec, *slot.Entity)
			content.AddChild(spec)
			if world.Components.Description.Has(*slot.Entity) {
				content.AddChild(styled.NewDescriptionText(world.Components.Description.Get(*slot.Entity).Description, res))
			}
		}
	} else {
		infoIdx := tabIndex - 1
		if infoIdx >= len(props.InfoTabs) {
			return nil
		}
		items := props.InfoTabs[infoIdx].Items
		if itemIndex >= len(items) {
			return nil
		}
		item := items[itemIndex]
		heading := item.Label
		if item.Value != "" {
			heading = fmt.Sprintf("%s  %s", item.Label, item.Value)
		}
		content.AddChild(styled.NewMenuText(heading, res))
		if item.Description != "" {
			content.AddChild(styled.NewDescriptionText(item.Description, res))
		}
		for _, d := range item.Details {
			if d.Value == "" {
				continue
			}
			content.AddChild(styled.NewDescriptionText(fmt.Sprintf("%s: %s", d.Label, d.Value), res))
		}
	}

	// タイトルバーは表示しない
	win := styled.NewSmallWindow(widget.NewContainer(), content)
	win.SetLocation(getCenterWinRect(world))
	return win
}
