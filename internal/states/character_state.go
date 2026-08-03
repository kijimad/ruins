package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
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
type characterSub string

const (
	charSubBrowse      characterSub = "browse"       // タブとカーソルの操作
	charSubEquipSelect characterSub = "equip_select" // 装備するアイテムの選択
)

// 画面タブ。装備と命令は編集可能、以降は読み取り専用の情報タブ
const (
	charScreenEquip   = 0 // 装備タブ
	charScreenCommand = 1 // 命令タブ。仲間の隊列ポリシーを編集する
)

// characterTabLabels は画面タブの見出し。編集可能な装備・命令の後ろに読み取り専用タブが並ぶ
var characterTabLabels = []string{"装備", "命令", "能力", "スキル", "効果", "健康", "基本"}

// CharacterState は画面タブメニューのステート。主人公と仲間で同じ画面を使い、対象を切り替えられる
type CharacterState struct {
	es.BaseState[w.World]
	target     ecs.Entity // 表示対象のキャラクター。ゼロ値なら主人公
	subState   characterSub
	detail     menuscreen.Detail // 詳細モーダルの表示状態とページ送り
	rebuild    bool              // 次フレームで UI を作り直すか
	mount      *hooks.Mount[characterProps]
	equipMount *hooks.Mount[charEquipProps]
	widget     *ebitenui.UI
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
	st.equipMount = hooks.NewMount[charEquipProps]()
	st.detail = menuscreen.NewDetail(st.detailContent)
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

	if st.detail.Active() {
		// 詳細表示中はページ送りと閉じるだけを扱い、通常のメニュー入力は止める
		if st.detail.HandleInput(world) {
			st.rebuild = true
		}
	} else if action, ok := st.handleInput(); ok {
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
	itemCounts := make([]int, 0, 2+len(props.InfoTabs))
	skips := make([][]bool, 0, 2+len(props.InfoTabs))
	itemCounts = append(itemCounts, len(props.EquipSlots))
	skips = append(skips, make([]bool, len(props.EquipSlots)))
	itemCounts = append(itemCounts, len(props.Commands))
	skips = append(skips, make([]bool, len(props.Commands)))
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
	menuDirty := st.mount.Update()
	equipDirty := st.equipMount.Update()
	if menuDirty || equipDirty || st.widget == nil || st.rebuild {
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

// dispatch は現在のサブステートにアクションを送る
func (st *CharacterState) dispatch(action inputmapper.ActionID) {
	switch st.subState {
	case charSubBrowse:
		st.mount.Dispatch(action)
	case charSubEquipSelect:
		st.equipMount.Dispatch(action)
	}
}

// handleInput はキー入力を Action に変換する。詳細モーダルの入力は Update 側で detail が扱う
func (st *CharacterState) handleInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	switch st.subState {
	case charSubBrowse, charSubEquipSelect:
		// 対象切替は閲覧中のみ。character 固有のキーなので共有入力ではなくここで読む。
		// 角括弧はキーボード配列で物理位置が変わり片方が別のキーコードになるため使わず、配列に依存しない , . を使う
		if st.subState == charSubBrowse {
			if ki.IsKeyJustPressed(ebiten.KeyComma) {
				return inputmapper.ActionMenuSubjectPrev, true
			}
			if ki.IsKeyJustPressed(ebiten.KeyPeriod) {
				return inputmapper.ActionMenuSubjectNext, true
			}
		}
		if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
			return inputmapper.ActionOpenItemDetail, true
		}
		return HandleMenuInput()
	}
	return "", false
}

// DoAction は Action を実行する
func (st *CharacterState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch st.subState {
	case charSubEquipSelect:
		return st.doEquipSelect(world, action)
	case charSubBrowse:
		return st.doBrowse(world, action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

func (st *CharacterState) doBrowse(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		// 命令タブには行ごとの詳細が無いので x を無視する
		if st.currentTabIndex() == charScreenCommand {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		st.detail.Open()
		st.rebuild = true
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSelect:
		return es.Transition[w.World]{Type: es.TransNone}, st.onBrowseSelect(world)
	case inputmapper.ActionMenuSubjectPrev:
		st.switchMember(world, -1)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSubjectNext:
		st.switchMember(world, 1)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// onBrowseSelect は閲覧中の Enter を処理する。装備タブは外す/装備選択、命令タブはポリシー変更や解雇、情報タブは詳細モーダルを開く
func (st *CharacterState) onBrowseSelect(world w.World) error {
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, characterMenuKey)
	props := st.mount.GetProps()
	switch menuState.TabIndex {
	case charScreenEquip:
		if menuState.ItemIndex >= len(props.EquipSlots) {
			return nil
		}
		slot := props.EquipSlots[menuState.ItemIndex]
		// 装備済みスロットは Enter で外す。空スロットは Enter で装備選択を開く
		if slot.Entity != nil {
			if err := st.unequipSlot(world, slot); err != nil {
				return err
			}
		} else {
			st.openEquipSelect(world, slot)
		}
		st.rebuild = true
	case charScreenCommand:
		if menuState.ItemIndex >= len(props.Commands) {
			return nil
		}
		row := props.Commands[menuState.ItemIndex]
		if row.Kind == cmdDismiss {
			st.dismissTarget(world)
		} else {
			st.cycleCommand(world, row.Kind)
		}
		st.rebuild = true
	default:
		// 情報タブは Enter で詳細モーダルを開く
		st.detail.Open()
		st.rebuild = true
	}
	return nil
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
		st.subState = charSubBrowse
		st.rebuild = true
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open()
		st.rebuild = true
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatch で処理される。装備選択は単一タブなのでタブ切替は何もしない
	default:
		return es.Transition[w.World]{}, fmt.Errorf("装備選択: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// openEquipSelect は空スロットに対して装備選択サブステートを開く
func (st *CharacterState) openEquipSelect(world w.World, slot equipItemData) {
	st.equipMount = hooks.NewMount[charEquipProps]()
	st.equipMount.SetProps(charEquipProps{
		Items:             equipableForSlot(world, slot.SlotNumber),
		SlotNumber:        slot.SlotNumber,
		PreviousEquipment: slot.Entity,
		TargetMember:      slot.Member,
	})
	st.subState = charSubEquipSelect
}

// unequipSlot は装備済みスロットのアイテムを外して持ち物へ戻す
func (st *CharacterState) unequipSlot(world w.World, slot equipItemData) error {
	if slot.Entity == nil {
		return nil
	}
	itemName := query.GetEntityName(*slot.Entity, world)
	if err := lifecycle.MoveToBackpack(world, *slot.Entity, slot.Member); err != nil {
		return err
	}
	st.logEquipChange(world, slot.Member, itemName, "を外した。")
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
	itemName := query.GetEntityName(item, world)

	if props.PreviousEquipment != nil {
		if err := lifecycle.MoveToBackpack(world, *props.PreviousEquipment, props.TargetMember); err != nil {
			return err
		}
	}
	lifecycle.MoveToEquip(world, item, props.TargetMember, props.SlotNumber)
	st.logEquipChange(world, props.TargetMember, itemName, "を装備した。")
	return nil
}

// logEquipChange は装備の着脱をゲームログに出す。対象キャラ名とアイテム名を添える
func (st *CharacterState) logEquipChange(world w.World, member ecs.Entity, itemName, verb string) {
	memberName := ""
	if world.ECS.Alive(member) && world.Components.Name.Has(member) {
		memberName = query.GetEntityName(member, world)
	}
	gamelog.New(query.GetGameLog(world)).
		Append(memberName).
		Append(" は ").
		ItemName(itemName).
		Append(" " + verb).
		Log()
}

// ================
// Props
// ================

type characterProps struct {
	TargetName  string // 表示対象のキャラクター名
	HasMultiple bool   // 切り替え可能な仲間がいるか
	EquipSlots  []equipItemData
	Commands    []commandRow    // 命令タブの隊列ポリシー。SquadAI を持つ仲間のみ行を持つ
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
		Commands:    fetchCommandRows(world, target),
		InfoTabs:    st.fetchInfoTabs(world, target),
	}
}

// ================
// 命令タブ
// ================

// commandKind は命令タブで編集する隊列ポリシーの種類
type commandKind string

const (
	cmdMovement     commandKind = "位置"
	cmdCombat       commandKind = "戦闘"
	cmdItemPickup   commandKind = "回収"
	cmdItemHandling commandKind = "処理"
	cmdSupply       commandKind = "補給"
	cmdDismiss      commandKind = "解雇"
)

// commandRow は命令タブの1行。種類と現在値を持つ。解雇の行は値を持たない
type commandRow struct {
	Kind  commandKind
	Value string
}

// fetchCommandRows は対象の隊列ポリシーを命令タブの行にする。SquadAI を持たない対象では空を返す
func fetchCommandRows(world w.World, target ecs.Entity) []commandRow {
	squad := query.GetSquadAI(world, target)
	if squad == nil {
		return nil
	}
	return []commandRow{
		{Kind: cmdMovement, Value: squad.Movement.String()},
		{Kind: cmdCombat, Value: squad.CombatCurrent.String()},
		{Kind: cmdItemPickup, Value: squad.ItemPickup.String()},
		{Kind: cmdItemHandling, Value: squad.ItemHandling.String()},
		{Kind: cmdSupply, Value: squad.Supply.String()},
		{Kind: cmdDismiss},
	}
}

// nextPolicy は候補列の中で cur の次の値を循環で返す
func nextPolicy[T comparable](all []T, cur T) T {
	for i, v := range all {
		if v == cur {
			return all[(i+1)%len(all)]
		}
	}
	if len(all) > 0 {
		return all[0]
	}
	return cur
}

// cycleCommand は対象の指定ポリシーを次の値へ進める。SquadAI のフィールドはポインタ経由で直接書き換える
func (st *CharacterState) cycleCommand(world w.World, kind commandKind) {
	squad := query.GetSquadAI(world, st.resolveTarget(world))
	if squad == nil {
		return
	}
	switch kind {
	case cmdMovement:
		squad.Movement = nextPolicy(gc.AllSquadMovements(), squad.Movement)
	case cmdCombat:
		squad.CombatCurrent = nextPolicy(gc.AllSquadCombatPolicies(), squad.CombatCurrent)
	case cmdItemPickup:
		squad.ItemPickup = nextPolicy(gc.AllItemPickupPolicies(), squad.ItemPickup)
	case cmdItemHandling:
		squad.ItemHandling = nextPolicy(gc.AllItemHandlingPolicies(), squad.ItemHandling)
	case cmdSupply:
		squad.Supply = nextPolicy(gc.AllSupplyPolicies(), squad.Supply)
	case cmdDismiss:
		// 解雇は値を巡回しない。onBrowseSelect が別に処理する
	}
}

// dismissTarget は現在の対象を解雇し、表示を主人公へ戻す
func (st *CharacterState) dismissTarget(world w.World) {
	target := st.resolveTarget(world)
	if err := lifecycle.DismissSquadMember(world, target); err != nil {
		return
	}
	st.target = ecs.Entity{}
}

// currentTabIndex は現在の画面タブの番号を返す
func (st *CharacterState) currentTabIndex() int {
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, characterMenuKey)
	return menuState.TabIndex
}

// buildCommandTable は命令タブの本文を組み立てる。ポリシー行と解雇行を並べ、対象が仲間でなければ案内を出す
func (st *CharacterState) buildCommandTable(rows []commandRow, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	container.AddChild(newPageIndicatorRow(itemIndex, len(rows), res))
	if len(rows) == 0 {
		container.AddChild(styled.NewDescriptionText("この対象に隊列指示はない", res))
		return container
	}
	columnWidths := []int{120, 160}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft}
	table := styled.NewTableContainer(columnWidths, res)
	for i, row := range rows {
		isSelected := i == itemIndex
		styled.NewTableRow(table, columnWidths, []string{string(row.Kind), row.Value}, aligns, &isSelected, res)
	}
	container.AddChild(table)
	return container
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

	// 見出しは対象キャラ名。仲間がいれば左右矢印で切替可能を示す。
	// 矢印は素の記号だとフォントに無く文字化けするため FontAwesome のアイコンを使う
	header := props.TargetName
	if props.HasMultiple {
		header = fmt.Sprintf("%s %s %s", consts.IconArrowLeft, props.TargetName, consts.IconArrowRight)
	}

	// コンテンツは現在タブの中身。装備は編集可能、以降は読み取り専用の情報タブ
	var content *widget.Container
	if tabIndex == charScreenEquip {
		content = st.buildEquipList(props.EquipSlots, itemIndex, res)
	} else if tabIndex == charScreenCommand {
		content = st.buildCommandTable(props.Commands, itemIndex, res)
	} else if infoIdx := tabIndex - 2; infoIdx >= 0 && infoIdx < len(props.InfoTabs) {
		content = st.buildInfoTable(props.InfoTabs[infoIdx], itemIndex, res)
	} else {
		content = widget.NewContainer()
	}

	extras := []string{"x 詳細"}
	if props.HasMultiple {
		extras = []string{", . 切替", "x 詳細"}
	}
	hint := menuNavHint(true, extras...)

	ui := newTabScreenUI(res, tabScreen{
		Header:    header,
		TabLabels: characterTabLabels,
		TabIndex:  tabIndex,
		Content:   content,
		Footer:    hint,
	})

	if st.subState == charSubEquipSelect {
		ui.AddWindow(st.buildEquipSelectWindow(world, res))
	}
	if st.detail.Active() {
		if win := st.detail.Window(world, getCenterWinRect(world)); win != nil {
			ui.AddWindow(win)
		}
	}

	return ui
}

// detailContent は現在の対象に応じた詳細内容を返す。詳細モーダルの唯一の定義点。
// 装備選択中は候補、閲覧中は装備中アイテム・空スロット・情報行を出し分ける。命令タブは詳細を持たない
func (st *CharacterState) detailContent(world w.World) (menuscreen.DetailContent, bool) {
	if st.subState == charSubEquipSelect {
		props := st.equipMount.GetProps()
		menuState, _ := hooks.GetState[hooks.TabMenuState](st.equipMount, "char_equip")
		if menuState.ItemIndex >= len(props.Items) {
			return menuscreen.DetailContent{}, false
		}
		return entityDetailContent(world, props.Items[menuState.ItemIndex]), true
	}

	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, characterMenuKey)
	props := st.mount.GetProps()
	switch menuState.TabIndex {
	case charScreenEquip:
		if menuState.ItemIndex >= len(props.EquipSlots) {
			return menuscreen.DetailContent{}, false
		}
		slot := props.EquipSlots[menuState.ItemIndex]
		if slot.Entity != nil {
			return entityDetailContent(world, *slot.Entity), true
		}
		// 空スロットは性能行を持たず、案内だけ出す。Rows を空で与え entity 解決を避ける
		return menuscreen.DetailContent{Name: slot.SlotLabel, Desc: "何も装備していない", Rows: []menuscreen.SpecRow{}}, true
	case charScreenCommand:
		return menuscreen.DetailContent{}, false
	default:
		infoIdx := menuState.TabIndex - 2
		if infoIdx < 0 || infoIdx >= len(props.InfoTabs) {
			return menuscreen.DetailContent{}, false
		}
		items := props.InfoTabs[infoIdx].Items
		if menuState.ItemIndex >= len(items) {
			return menuscreen.DetailContent{}, false
		}
		return infoDetailContent(items[menuState.ItemIndex]), true
	}
}

// entityDetailContent はエンティティの名前・説明・性能を詳細内容にする
func entityDetailContent(world w.World, entity ecs.Entity) menuscreen.DetailContent {
	name := ""
	if world.Components.Name.Has(entity) {
		name = world.Components.Name.Get(entity).Name
	}
	desc := ""
	if world.Components.Description.Has(entity) {
		desc = world.Components.Description.Get(entity).Description
	}
	return menuscreen.DetailContent{Name: name, Desc: desc, Entity: entity}
}

// infoDetailContent は情報タブの1行を詳細内容にする。見出しと説明、内訳の行を出す
func infoDetailContent(item statusItemData) menuscreen.DetailContent {
	heading := item.Label
	if item.Value != "" {
		heading = fmt.Sprintf("%s  %s", item.Label, item.Value)
	}
	rows := []menuscreen.SpecRow{}
	for _, d := range item.Details {
		if d.Value == "" {
			continue
		}
		rows = append(rows, menuscreen.SpecRow{Label: d.Label, Value: d.Value})
	}
	return menuscreen.DetailContent{Name: heading, Desc: item.Description, Rows: rows}
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
