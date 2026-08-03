package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 画面タブメニュー。キャラクター情報の閲覧・操作を装備・スキルのタブでまとめる。
// 所持アイテムへの動詞でなくキャラクター情報が対象なので、動詞タブ画面とは別 state にする。
// ダンジョンからの直達ショートカットは持たず、ダンジョンメニューから開く。

const characterMenuKey = "character"

// charTab は画面タブの種別。装備と命令は編集可能、以降は読み取り専用の情報タブ
type charTab string

const (
	charTabEquip   charTab = "equip"   // 装備。編集可能
	charTabCommand charTab = "command" // 命令。仲間の隊列ポリシーを編集する
	charTabAbility charTab = "ability" // 能力
	charTabSkill   charTab = "skill"   // スキル
	charTabEffect  charTab = "effect"  // 効果
	charTabHealth  charTab = "health"  // 健康
	charTabBasic   charTab = "basic"   // 基本
)

// characterTabs はタブの種別と見出しを表示順に対応づける。タブ番号はこの並び順で決まる。
// 編集可能な装備・命令の後ろに読み取り専用の情報タブが並ぶ
var characterTabs = []struct {
	Kind  charTab
	Label string
}{
	{charTabEquip, "装備"},
	{charTabCommand, "命令"},
	{charTabAbility, "能力"},
	{charTabSkill, "スキル"},
	{charTabEffect, "効果"},
	{charTabHealth, "健康"},
	{charTabBasic, "基本"},
}

// charFirstInfoTab は情報タブが始まるタブ番号。編集タブの装備・命令の後に情報タブが並ぶ。
// 先頭の情報タブ ability の位置から導き、編集タブを増減しても追従させる。マジックナンバーを避ける
var charFirstInfoTab = indexOfCharTab(charTabAbility)

// indexOfCharTab はタブ種別の表示順での位置を返す。見つからなければ 0
func indexOfCharTab(kind charTab) int {
	for i, t := range characterTabs {
		if t.Kind == kind {
			return i
		}
	}
	return 0
}

// charTabAt はタブ番号に対応するタブ種別を返す。範囲外は空文字を返す
func charTabAt(index int) charTab {
	if index < 0 || index >= len(characterTabs) {
		return ""
	}
	return characterTabs[index].Kind
}

// characterTabLabels はタブの見出し一覧を表示順で返す
func characterTabLabels() []string {
	labels := make([]string, len(characterTabs))
	for i, t := range characterTabs {
		labels[i] = t.Label
	}
	return labels
}

// CharacterState は画面タブメニューのステート。主人公と仲間で同じ画面を使い、対象を切り替えられる
type CharacterState struct {
	es.BaseState[w.World]
	target ecs.Entity            // 表示対象のキャラクター。ゼロ値なら主人公
	detail menuscreen.Detail     // 詳細モーダル。overlay として Screen に登録する
	equip  characterEquipOverlay // 装備選択。overlay として Screen に登録する
	screen Screen[characterProps]
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
	st.detail = menuscreen.NewDetail(st.detailContent)
	st.equip = newCharacterEquipOverlay(&st.detail)
	// detail を equip より前に登録する。装備選択中に x で開いた詳細が入力を優先する
	st.screen = NewScreen[characterProps](&st.detail, &st.equip)
	st.screen.WithSystems(&gs.StatsChangedSystem{}, &gs.WeightDirtySystem{})
	return nil
}

// Update はステートの更新処理を Screen へ委譲する
func (st *CharacterState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world, st)
}

// Draw はステートの描画を Screen へ委譲する
func (st *CharacterState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// HandleInput は閲覧中のキー入力を Action に変換する。装備選択中は overlay が入力を専有するため
// Screen はこれを呼ばない。対象切替の , . と詳細の x は画面固有キーとしてここで読む
func (st *CharacterState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyComma) {
		return inputmapper.ActionMenuSubjectPrev, true
	}
	if ki.IsKeyJustPressed(ebiten.KeyPeriod) {
		return inputmapper.ActionMenuSubjectNext, true
	}
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return HandleMenuInput()
}

// DoAction は閲覧中の Action を実行する
func (st *CharacterState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		// 命令タブには行ごとの詳細が無いので x を無視する
		if charTabAt(st.screen.Selection().TabIndex) == charTabCommand {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		st.detail.Open()
		st.screen.MarkDirty()
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

// onBrowseSelect は閲覧中の Enter を処理する。装備タブは外す・装備選択、命令タブはポリシー変更や解雇、情報タブは詳細を開く
func (st *CharacterState) onBrowseSelect(world w.World) error {
	sel := st.screen.Selection()
	props := st.screen.Props()
	switch charTabAt(sel.TabIndex) {
	case charTabEquip:
		if sel.ItemIndex >= len(props.EquipSlots) {
			return nil
		}
		slot := props.EquipSlots[sel.ItemIndex]
		// 装備済みスロットは Enter で外す。空スロットは Enter で装備選択を開く
		if slot.Entity != nil {
			if err := st.unequipSlot(world, slot); err != nil {
				return err
			}
		} else {
			st.equip.Open(world, slot)
		}
		st.screen.MarkDirty()
	case charTabCommand:
		if sel.ItemIndex >= len(props.Commands) {
			return nil
		}
		row := props.Commands[sel.ItemIndex]
		if row.Kind == cmdDismiss {
			st.dismissTarget(world)
		} else {
			st.cycleCommand(world, row.Kind)
		}
		st.screen.MarkDirty()
	default:
		st.detail.Open()
		st.screen.MarkDirty()
	}
	return nil
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
	logEquipChange(world, slot.Member, itemName, "を外した。")
	return nil
}

// logEquipChange は装備の着脱をゲームログに出す。対象キャラ名とアイテム名を添える
func logEquipChange(world w.World, member ecs.Entity, itemName, verb string) {
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

// switchMember は表示対象を dir 方向の隣のキャラへ巡回で切り替える
func (st *CharacterState) switchMember(world w.World, dir int) {
	members := characterMembers(world)
	if len(members) <= 1 {
		return
	}
	cur := st.resolveTarget(world)
	// resolveTarget は主人公か生存メンバーを返すため cur は必ず members に含まれる。
	// 万一含まれなくても idx は 0 のまま、すなわち主人公を起点に巡回する
	idx := 0
	for i, m := range members {
		if m == cur {
			idx = i
			break
		}
	}
	st.target = members[(idx+dir+len(members))%len(members)]
	st.screen.MarkDirty()
}

// fetch は表示対象のスナップショットを組む
func (st *CharacterState) fetch(world w.World) characterProps {
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

// menu は装備・命令・情報タブのカーソル構成を返す。情報タブの見出し行はカーソルを飛ばす
func (st *CharacterState) menu(props characterProps) MenuConfig {
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
	return MenuConfig{Key: characterMenuKey, TabCount: len(itemCounts), ItemCounts: itemCounts, Skips: skips}
}

// view は現在タブの本体を純粋描画へ委譲する。overlay の窓は Screen が重ねる
func (st *CharacterState) view(_ w.World, props characterProps, sel Selection, res resources.UIResources) *ebitenui.UI {
	return buildCharacterUI(props, sel, res)
}

// detailContent は現在の対象に応じた詳細内容を返す。詳細モーダルの唯一の定義点。
// 装備選択中は候補、閲覧中は装備中アイテム・空スロット・情報行を出し分ける。命令タブは詳細を持たない
func (st *CharacterState) detailContent(world w.World) (menuscreen.DetailContent, bool) {
	if st.equip.Active() {
		item, ok := st.equip.selectedItem()
		if !ok {
			return menuscreen.DetailContent{}, false
		}
		return entityDetailContent(world, item), true
	}

	sel := st.screen.Selection()
	props := st.screen.Props()
	switch charTabAt(sel.TabIndex) {
	case charTabEquip:
		if sel.ItemIndex >= len(props.EquipSlots) {
			return menuscreen.DetailContent{}, false
		}
		slot := props.EquipSlots[sel.ItemIndex]
		if slot.Entity != nil {
			return entityDetailContent(world, *slot.Entity), true
		}
		// 空スロットは性能行を持たず、案内だけ出す。Rows を空で与え entity 解決を避ける
		return menuscreen.DetailContent{Name: slot.SlotLabel, Desc: "何も装備していない", Rows: []menuscreen.SpecRow{}}, true
	case charTabCommand:
		return menuscreen.DetailContent{}, false
	default:
		infoIdx := sel.TabIndex - charFirstInfoTab
		if infoIdx < 0 || infoIdx >= len(props.InfoTabs) {
			return menuscreen.DetailContent{}, false
		}
		items := props.InfoTabs[infoIdx].Items
		if sel.ItemIndex >= len(items) {
			return menuscreen.DetailContent{}, false
		}
		return infoDetailContent(items[sel.ItemIndex]), true
	}
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

// buildCommandTable は命令タブの本文を組み立てる。ポリシー行と解雇行を並べ、対象が仲間でなければ案内を出す
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

// buildUI は現在の state から描画に必要なデータを取り出し、純粋描画の buildCharacterUI へ渡す。
// 詳細と装備選択のオーバーレイ窓は world とコントローラを要するため、ここで重ねる

// detailContent は現在の対象に応じた詳細内容を返す。詳細モーダルの唯一の定義点。
// 装備選択中は候補、閲覧中は装備中アイテム・空スロット・情報行を出し分ける。命令タブは詳細を持たない

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
