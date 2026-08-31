package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 画面タブメニュー。キャラクター情報の閲覧・操作を装備・スキルのタブでまとめる。
// 所持アイテムへの動詞でなくキャラクター情報が対象なので、動詞タブ画面とは別 state にする。
// ダンジョンからの直達ショートカットは持たず、ダンジョンメニューから開く。

const characterMenuKey = "character"

// charTab は画面タブの種別。装備は編集可能、以降は読み取り専用の情報タブ
type charTab string

const (
	charTabEquip   charTab = "equip"   // 装備。編集可能
	charTabAbility charTab = "ability" // 能力
	charTabSkill   charTab = "skill"   // スキル
	charTabEffect  charTab = "effect"  // 効果
	charTabHealth  charTab = "health"  // 健康
	charTabBasic   charTab = "basic"   // 基本
)

// characterTabs はタブの種別と見出しを表示順に対応づける。タブ番号はこの並び順で決まる。
// 編集可能な装備の後ろに読み取り専用の情報タブが並ぶ
var characterTabs = []struct {
	Kind  charTab
	Label string
}{
	{charTabEquip, "Equipment"},
	{charTabAbility, "Abilities"},
	{charTabSkill, "Skills"},
	{charTabEffect, "Effects"},
	{charTabHealth, "Health"},
	{charTabBasic, "Basic"},
}

// charFirstInfoTab は情報タブが始まるタブ番号。編集タブの装備の後に情報タブが並ぶ。
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

// characterTabLabels はタブの見出し一覧を現在言語で表示順に返す
func characterTabLabels(world w.World) []string {
	labels := make([]string, len(characterTabs))
	for i, t := range characterTabs {
		labels[i] = query.T(world, t.Label)
	}
	return labels
}

// CharacterState は画面タブメニューのステート。主人公の装備と情報を閲覧・編集する
type CharacterState struct {
	es.BaseState[w.World]
	detail overlay.Detail        // 詳細モーダル。overlay として Screen に登録する
	equip  characterEquipOverlay // 装備選択。overlay として Screen に登録する
	screen *menuloop.Screen[CharacterProps]
}

var _ es.State[w.World] = &CharacterState{}
var _ menuloop.KeyBindings = &CharacterState{}

// OnStart はステートが開始される際に呼ばれる
func (st *CharacterState) OnStart(_ w.World) error {
	st.detail = overlay.NewDetail(st.detailContent)
	st.equip = newCharacterEquipOverlay(&st.detail)
	// detail を equip より前に登録する。装備選択中に x で開いた詳細が入力を優先する
	st.screen = menuloop.NewScreen[CharacterProps](st, &st.detail, &st.equip)
	return nil
}

// Update はステートの更新処理を Screen へ委譲する
func (st *CharacterState) Update(world w.World) (es.Transition[w.World], error) {
	// 装備の着脱でステータスと重量が変わる。再計算を回して表示を更新する
	if err := runUpdaters(world, &gs.StatsChangedSystem{}, &gs.WeightDirtySystem{}); err != nil {
		return es.Transition[w.World]{}, err
	}
	return st.screen.Update(world)
}

// Draw はステートの描画を Screen へ委譲する
func (st *CharacterState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// KeyBindings は共通入力に加える独自キーの束縛表。x で選択中の詳細モーダルを開く。
// 装備選択中は overlay が入力を専有するため Screen はこれを読まない
func (st *CharacterState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
}

// DoAction は閲覧中の Action を実行する
func (st *CharacterState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSelect:
		return es.Transition[w.World]{Type: es.TransNone}, st.onBrowseSelect(world)
	default:
		return es.Transition[w.World]{}, fmt.Errorf("unknown action: %s", action)
	}
}

// onBrowseSelect は閲覧中の Enter を処理する。装備タブは外す・装備選択、情報タブは詳細を開く
func (st *CharacterState) onBrowseSelect(world w.World) error {
	cursor := st.screen.Selection()
	props := st.screen.Props()
	switch charTabAt(cursor.TabIndex) {
	case charTabEquip:
		if cursor.ItemIndex >= len(props.EquipSlots) {
			return nil
		}
		slot := props.EquipSlots[cursor.ItemIndex]
		// 装備済みスロットは Enter で外す。空スロットは Enter で装備選択を開く
		if slot.Entity != nil {
			if err := st.unequipSlot(world, slot); err != nil {
				return err
			}
		} else {
			st.equip.Open(world, slot)
		}
	default:
		st.detail.Open(world)
	}
	return nil
}

// unequipSlot は装備済みスロットのアイテムを外して持ち物へ戻す
func (st *CharacterState) unequipSlot(world w.World, slot equipItemData) error {
	if slot.Entity == nil {
		return nil
	}
	itemName := query.GetEntityName(*slot.Entity, world)
	if err := lifecycle.MoveToBackpack(world, *slot.Entity, slot.Character); err != nil {
		return err
	}
	logEquipChange(world, slot.Character, itemName, query.T(world, "%s unequipped %s."))
	return nil
}

// logEquipChange は装備の着脱をゲームログに出す。format は対象キャラ名とアイテム名を差し込む
// "%s ... %s" 形式の翻訳済み書式。アイテム名はシアンを保つ
func logEquipChange(world w.World, character ecs.Entity, itemName, format string) {
	characterName := ""
	if world.ECS.Alive(character) && world.Components.Name.Has(character) {
		characterName = query.GetEntityName(character, world)
	}
	gamelog.New(query.GetGameLog(world)).
		Markup(fmt.Sprintf(format, characterName, gamelog.Tag("item", itemName))).
		Log()
}

// Fetch は主人公のスナップショットを組む
func (st *CharacterState) Fetch(world w.World) (CharacterProps, error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return CharacterProps{}, err
	}
	// 死亡直後のフレームは実体が消えている。遷移までの過渡状態なので空表示にする
	if !world.ECS.Alive(player) {
		return CharacterProps{}, nil
	}
	name := ""
	if world.Components.Name.Has(player) {
		name = query.GetEntityName(player, world)
	}
	return CharacterProps{
		TargetName: name,
		EquipSlots: characterEquipSlots(world, player),
		InfoTabs:   st.fetchInfoTabs(world, player),
	}, nil
}

// Menu は装備・情報タブのカーソル構成を返す。情報タブの見出し行はカーソルを飛ばす
func (st *CharacterState) Menu(props CharacterProps) menuloop.MenuConfig {
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
	return menuloop.MenuConfig{Key: characterMenuKey, TabCount: len(itemCounts), ItemCounts: itemCounts, Skips: skips}
}

// detailContent は現在の対象に応じた詳細内容を返す。詳細モーダルの唯一の定義点。
// 装備選択中は候補、閲覧中は装備中アイテム・空スロット・情報行を出し分ける
func (st *CharacterState) detailContent(world w.World) (overlay.DetailContent, bool) {
	if st.equip.Active() {
		item, ok := st.equip.selectedItem()
		if !ok {
			return overlay.DetailContent{}, false
		}
		return overlay.EntityDetailContent(world, item), true
	}

	cursor := st.screen.Selection()
	props := st.screen.Props()
	switch charTabAt(cursor.TabIndex) {
	case charTabEquip:
		if cursor.ItemIndex >= len(props.EquipSlots) {
			return overlay.DetailContent{}, false
		}
		slot := props.EquipSlots[cursor.ItemIndex]
		if slot.Entity != nil {
			return overlay.EntityDetailContent(world, *slot.Entity), true
		}
		// 空スロットは性能行を持たず案内だけ出す
		return overlay.DetailContent{Name: slot.SlotLabel, Desc: query.T(world, "Nothing equipped")}, true
	default:
		infoIdx := cursor.TabIndex - charFirstInfoTab
		if infoIdx < 0 || infoIdx >= len(props.InfoTabs) {
			return overlay.DetailContent{}, false
		}
		items := props.InfoTabs[infoIdx].Items
		if cursor.ItemIndex >= len(items) {
			return overlay.DetailContent{}, false
		}
		return infoDetailContent(items[cursor.ItemIndex]), true
	}
}

// ================
// Props
// ================

// CharacterProps は画面の表示 props。menuloop.Screen の型引数として渡す
type CharacterProps struct {
	TargetName string          // 表示対象のキャラクター名
	EquipSlots []equipItemData //
	InfoTabs   []statusTabData // 能力・スキル・効果・健康・基本の読み取り専用タブ
}

// equipItemData は装備スロット1つ分の表示データ
type equipItemData struct {
	SlotLabel  string
	ItemName   string
	SlotNumber gc.EquipmentSlotNumber
	Entity     *ecs.Entity // 装備中エンティティ。空きなら nil
	Character  ecs.Entity  // 装備スロットを持つキャラ本体
}

// charEquipProps は装備選択の Props
type charEquipProps struct {
	Items             []ecs.Entity
	SlotNumber        gc.EquipmentSlotNumber
	PreviousEquipment *ecs.Entity
	TargetCharacter   ecs.Entity
}

// characterEquipSlots は指定したキャラクターの全装備スロットを列挙する
func characterEquipSlots(world w.World, character ecs.Entity) []equipItemData {
	items := make([]equipItemData, 0, 12)

	weapons := query.GetWeapons(world, character)
	weaponSlots := []gc.EquipmentSlotNumber{gc.SlotWeapon1, gc.SlotWeapon2, gc.SlotWeapon3, gc.SlotWeapon4, gc.SlotWeapon5}
	for i, weapon := range weapons {
		name := ""
		if weapon != nil {
			name = query.GetEntityName(*weapon, world)
		}
		items = append(items, equipItemData{SlotLabel: query.T(world, "Weapon %d", i+1), ItemName: name, SlotNumber: weaponSlots[i], Entity: weapon, Character: character})
	}

	// armor・armorLabels・armorSlots は防具7スロットに対応する並行配列。
	// GetArmorEquipments はスロット数ぶんの固定長スライスを返し、3者の長さは常に一致する
	armor := query.GetArmorEquipments(world, character)
	armorLabels := []string{query.T(world, "Armor (head)"), query.T(world, "Armor (torso)"), query.T(world, "Armor (arms)"), query.T(world, "Armor (hands)"), query.T(world, "Armor (legs)"), query.T(world, "Armor (feet)"), query.T(world, "Armor (jewelry)")}
	armorSlots := []gc.EquipmentSlotNumber{gc.SlotHead, gc.SlotTorso, gc.SlotArms, gc.SlotHands, gc.SlotLegs, gc.SlotFeet, gc.SlotJewelry}
	for i, slot := range armor {
		name := ""
		if slot != nil {
			name = query.GetEntityName(*slot, world)
		}
		items = append(items, equipItemData{SlotLabel: armorLabels[i], ItemName: name, SlotNumber: armorSlots[i], Entity: slot, Character: character})
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
			if !query.IsWeapon(world, entity) {
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

// infoDetailContent は情報タブの1行を詳細内容にする。見出しと説明、内訳の行を出す
func infoDetailContent(item statusItemData) overlay.DetailContent {
	rows := []entityspec.SpecRow{}
	sectioned := false
	for _, d := range item.Details {
		if d.Header {
			sectioned = true
		}
		if d.Value == "" && !d.Header {
			continue
		}
		rows = append(rows, entityspec.SpecRow{Label: d.Label, Value: d.Value, Header: d.Header})
	}

	// 見出しに値を重ねるのは内訳がフラットなときだけにする。
	// 内訳が見出し行を持つなら値はそちらに出るので、見出しはラベルだけにして重複を避ける
	heading := item.Label
	if item.Value != "" && !sectioned {
		heading = fmt.Sprintf("%s  %s", item.Label, item.Value)
	}
	return overlay.DetailContent{Name: heading, Desc: item.Description, Rows: rows}
}
