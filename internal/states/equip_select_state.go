package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// EquipSelectState はスロット1つの装備を選ぶメニュー。装備タブでスロットを選ぶと push される。
// 装備済みなら一覧の先頭に「外す」が並ぶ。x の詳細は現装備と候補を2枚並べて比較する。
type EquipSelectState struct {
	es.BaseState[w.World]
	slotNumber        gc.EquipmentSlotNumber
	targetCharacter   ecs.Entity
	previousEquipment *ecs.Entity // 装備中エンティティ。空きなら nil
	detail            overlay.Detail
	screen            *menuloop.Screen[EquipSelectProps]
}

var _ es.State[w.World] = &EquipSelectState{}
var _ menuloop.KeyBindings = &EquipSelectState{}

// EquipSelectProps は装備選択の表示 props
type EquipSelectProps struct {
	Items             []ecs.Entity
	SlotNumber        gc.EquipmentSlotNumber
	PreviousEquipment *ecs.Entity
	TargetCharacter   ecs.Entity
}

// newEquipSelectState はスロットに対する装備選択を開くファクトリを返す
func newEquipSelectState(slot equipItemData) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &EquipSelectState{
			slotNumber:        slot.SlotNumber,
			targetCharacter:   slot.Character,
			previousEquipment: slot.Entity,
		}, nil
	}
}

// OnStart はステートが開始される際に呼ばれる
func (st *EquipSelectState) OnStart(_ w.World) error {
	st.detail = overlay.NewDetail(st.detailContent)
	// 候補にカーソルがあるとき、詳細を現装備と候補の2枠で見せる
	st.detail.SetCompare(st.compareContent)
	st.screen = menuloop.NewScreen[EquipSelectProps](st, &st.detail)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *EquipSelectState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *EquipSelectState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// KeyBindings は x の詳細表示を共通入力に足す
func (st *EquipSelectState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
}

// DoAction はActionを実行する。選択で装着または外して閉じる
func (st *EquipSelectState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuSelect:
		if choice, ok := st.selection(); ok {
			if err := applyEquipChoice(world, choice, st.slotNumber, st.targetCharacter, st.previousEquipment); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
		return es.Transition[w.World]{Type: es.TransPop}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("equipSelect: unsupported action: %s", action)
	}
}

// Fetch は世界から表示 props を構築する
func (st *EquipSelectState) Fetch(world w.World) (EquipSelectProps, error) {
	return EquipSelectProps{
		Items:             equipableForSlot(world, st.slotNumber),
		SlotNumber:        st.slotNumber,
		PreviousEquipment: st.previousEquipment,
		TargetCharacter:   st.targetCharacter,
	}, nil
}

// Menu は単一リストの構成を返す。装備済みなら先頭の「外す」ぶんを1つ足す
func (st *EquipSelectState) Menu(props EquipSelectProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: "equip_select", TabCount: 1, ItemCounts: []int{equipChoiceCount(props)}, ItemsPerPage: menuloop.ItemsPerPageAuto}
}

// ViewUI は候補一覧を中央パネルへ組む
func (st *EquipSelectState) ViewUI(world w.World, props EquipSelectProps, cursor menuloop.Selection, res resources.UIResources) uicore.Drawable {
	return buildEquipSelectUI(world, props, cursor.ItemIndex, res)
}

// selection は現在カーソルが指す選択を返す
func (st *EquipSelectState) selection() (equipChoice, bool) {
	return equipChoiceAt(st.screen.Props(), st.screen.Selection().ItemIndex)
}

// detailContent は主たる詳細内容を返す。候補ならその性能、「外す」なら外す対象の性能を出す
func (st *EquipSelectState) detailContent(world w.World) (overlay.DetailContent, bool) {
	choice, ok := st.selection()
	if !ok {
		return overlay.DetailContent{}, false
	}
	if choice.unequip {
		if st.previousEquipment == nil {
			return overlay.DetailContent{}, false
		}
		return overlay.EntityDetailContent(world, *st.previousEquipment), true
	}
	return overlay.EntityDetailContent(world, choice.entity), true
}

// compareContent は詳細モーダルの左枠に並べる現装備を返す。候補にカーソルがあり装備済みの
// ときだけ ok を返し、現装備と候補を2枠で見せる。空スロットや「外す」では ok=false で1枠に戻る
func (st *EquipSelectState) compareContent(world w.World) (overlay.DetailContent, bool) {
	choice, ok := st.selection()
	if !ok || choice.unequip || st.previousEquipment == nil {
		return overlay.DetailContent{}, false
	}
	return overlay.EntityDetailContent(world, *st.previousEquipment), true
}

// equipChoice はカーソルが指す装備選択。unequip なら「外す」、そうでなければ entity が候補
type equipChoice struct {
	unequip bool
	entity  ecs.Entity
}

// equipChoiceAt は候補一覧の index を選択へ写す。装備済みなら先頭が「外す」で、候補は1つ後ろへ詰まる
func equipChoiceAt(props EquipSelectProps, index int) (equipChoice, bool) {
	if props.PreviousEquipment != nil {
		if index == 0 {
			return equipChoice{unequip: true}, true
		}
		index--
	}
	if index < 0 || index >= len(props.Items) {
		return equipChoice{}, false
	}
	return equipChoice{entity: props.Items[index]}, true
}

// equipChoiceCount は選択できる項目数を返す。装備済みなら先頭の「外す」を1つ足す
func equipChoiceCount(props EquipSelectProps) int {
	if props.PreviousEquipment != nil {
		return len(props.Items) + 1
	}
	return len(props.Items)
}

// applyEquipChoice は選択を実行する。「外す」なら現装備を外し、候補なら装着する。
// 既存の装備があれば持ち物へ戻す
func applyEquipChoice(world w.World, choice equipChoice, slotNumber gc.EquipmentSlotNumber, target ecs.Entity, previous *ecs.Entity) error {
	if choice.unequip {
		if previous == nil {
			return nil
		}
		itemName := query.GetEntityName(*previous, world)
		if err := lifecycle.MoveToBackpack(world, *previous, target); err != nil {
			return err
		}
		logEquipChange(world, target, itemName, query.T(world, "%s unequipped %s."))
		return nil
	}
	itemName := query.GetEntityName(choice.entity, world)
	if previous != nil {
		if err := lifecycle.MoveToBackpack(world, *previous, target); err != nil {
			return err
		}
	}
	lifecycle.MoveToEquip(world, choice.entity, target, slotNumber)
	logEquipChange(world, target, itemName, query.T(world, "%s equipped %s."))
	return nil
}

// buildEquipSelectUI はアイコン付きの候補一覧を中央パネルへ組む。装備済みなら先頭に「外す」が並ぶ
func buildEquipSelectUI(world w.World, props EquipSelectProps, selectedIndex int, res resources.UIResources) uicore.Drawable {
	var rows []menuframe.Row
	if props.PreviousEquipment != nil {
		rows = append(rows, menuframe.Row{Cells: []styled.Cell{styled.IconCell(nil), styled.TextCell(query.T(world, "Unequip"))}})
	}
	for _, entity := range props.Items {
		rows = append(rows, menuframe.Row{Cells: []styled.Cell{styled.IconCell(menuIcon(world, entity)), styled.TextCell(query.GetEntityName(entity, world))}})
	}
	list, pager := menuframe.RenderList(selectedIndex, rows, styled.Cols(styled.Icon(), styled.Name()),
		menuframe.ListOpts{EmptyText: query.T(world, "Nothing to equip")}, res)
	return menuframe.PanelScreen(world, res, query.T(world, "Choose equipment"), list, keybind.HelpHint(world), pager)
}
