package states

import (
	"fmt"
	"image"
	"strconv"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// characterEquipOverlay は装備選択を Overlay として実装する。第2のカーソル mount を自前で持ち、
// 装備選択という変則挙動を Screen 本体でなくこの overlay に閉じ込める。x で詳細モーダルを
// 入れ子に開くため、共有の Detail への参照を持つ
type characterEquipOverlay struct {
	active bool
	mount  *hooks.Mount[charEquipProps]
	detail *overlay.Detail
}

var _ overlay.Layer = (*characterEquipOverlay)(nil)
var _ overlay.ScreenRenderer = (*characterEquipOverlay)(nil)

// newCharacterEquipOverlay は共有の詳細モーダルを参照する装備選択 overlay を作る
func newCharacterEquipOverlay(detail *overlay.Detail) characterEquipOverlay {
	return characterEquipOverlay{mount: hooks.NewMount[charEquipProps](), detail: detail}
}

// Active は装備選択中かを返す
func (o *characterEquipOverlay) Active() bool { return o.active }

// Open はスロットに対する装備選択を開き、候補を読み込む。
// 装備済みスロットなら PreviousEquipment に現装備を持ち、候補一覧の先頭に「外す」が並ぶ
func (o *characterEquipOverlay) Open(world w.World, slot equipItemData) {
	// 開くたびに mount を作り直し、再オープン時はカーソルを先頭へ戻す
	o.mount = hooks.NewMount[charEquipProps]()
	o.mount.SetProps(charEquipProps{
		Items:             equipableForSlot(world, slot.SlotNumber),
		SlotNumber:        slot.SlotNumber,
		PreviousEquipment: slot.Entity,
		TargetCharacter:   slot.Character,
	})
	o.active = true
}

// equipChoice はカーソルが指す装備選択。unequip なら「外す」、そうでなければ entity が候補
type equipChoice struct {
	unequip bool
	entity  ecs.Entity
}

// equipChoiceAt は候補一覧の index を選択へ写す。装備済みなら先頭が「外す」で、候補は1つ後ろへ詰まる
func equipChoiceAt(props charEquipProps, index int) (equipChoice, bool) {
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

// selection は現在カーソルが指す選択を返す
func (o *characterEquipOverlay) selection() (equipChoice, bool) {
	ms, _ := hooks.GetState[hooks.TabMenuState](o.mount, "char_equip")
	return equipChoiceAt(o.mount.GetProps(), ms.ItemIndex)
}

// selectedItem は現在カーソルが指す候補を返す。「外す」や候補外なら ok=false
func (o *characterEquipOverlay) selectedItem() (ecs.Entity, bool) {
	choice, ok := o.selection()
	if !ok || choice.unequip {
		return gc.InvalidEntity, false
	}
	return choice.entity, true
}

// HandleInput は装備選択中の入力を処理する。毎フレーム呼ばれるので自前カーソル mount の維持もここで行う。
// x で詳細を入れ子に開き、Enter で装着して閉じ、Esc で閉じる
func (o *characterEquipOverlay) HandleInput(world w.World) error {
	if !o.active {
		return nil
	}
	props := o.mount.GetProps()
	hooks.UseTabMenu(o.mount.Store(), "char_equip", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{equipChoiceCount(props)},
	})

	if action, ok := keybind.ReadInput(world, equipSelectTable); ok {
		switch action {
		case inputmapper.ActionOpenItemDetail:
			o.detail.Open(world)
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			o.active = false
		case inputmapper.ActionMenuSelect:
			if err := o.execute(world); err != nil {
				return err
			}
			o.active = false
		default:
			// 移動系は自前カーソルの mount が消費する。それ以外は装備選択中は扱わない
			o.mount.DispatchNav(action)
		}
	}
	o.mount.Update()
	return nil
}

// execute は現在の選択を実行する。「外す」なら現装備を外し、候補なら装着する。
// 既存の装備があれば持ち物へ戻す
func (o *characterEquipOverlay) execute(world w.World) error {
	choice, ok := o.selection()
	if !ok {
		return nil
	}
	if choice.unequip {
		return o.unequip(world)
	}
	props := o.mount.GetProps()
	itemName := query.GetEntityName(choice.entity, world)
	if props.PreviousEquipment != nil {
		if err := lifecycle.MoveToBackpack(world, *props.PreviousEquipment, props.TargetCharacter); err != nil {
			return err
		}
	}
	lifecycle.MoveToEquip(world, choice.entity, props.TargetCharacter, props.SlotNumber)
	logEquipChange(world, props.TargetCharacter, itemName, query.T(world, "%s equipped %s."))
	return nil
}

// unequip は現装備を外して持ち物へ戻す。外す対象が無ければ何もしない
func (o *characterEquipOverlay) unequip(world w.World) error {
	props := o.mount.GetProps()
	if props.PreviousEquipment == nil {
		return nil
	}
	itemName := query.GetEntityName(*props.PreviousEquipment, world)
	if err := lifecycle.MoveToBackpack(world, *props.PreviousEquipment, props.TargetCharacter); err != nil {
		return err
	}
	logEquipChange(world, props.TargetCharacter, itemName, query.T(world, "%s unequipped %s."))
	return nil
}

// RenderOverlay は装備候補のモーダルを uicore のツリーへ組む。Screen が本体の上へ重ねる。
func (o *characterEquipOverlay) RenderOverlay(world w.World, _ image.Rectangle) uicore.Drawable {
	if !o.active {
		return nil
	}
	props := o.mount.GetProps()
	ms, _ := hooks.GetState[hooks.TabMenuState](o.mount, "char_equip")
	return buildEquipSelectUI(world, props, ms.ItemIndex, world.Resources.UIResources)
}

// equipChoiceCount は選択できる項目数を返す。装備済みなら先頭の「外す」を1つ足す
func equipChoiceCount(props charEquipProps) int {
	if props.PreviousEquipment != nil {
		return len(props.Items) + 1
	}
	return len(props.Items)
}

// buildEquipSelectUI はアイコン付きの候補一覧を中央パネルへ組む。装備済みなら先頭に「外す」が並ぶ
func buildEquipSelectUI(world w.World, props charEquipProps, selectedIndex int, res resources.UIResources) uicore.Drawable {
	var rows []menuframe.Row
	if props.PreviousEquipment != nil {
		rows = append(rows, menuframe.Row{Cells: []styled.Cell{styled.IconCell(nil), styled.TextCell(query.T(world, "Unequip"))}})
	}
	for _, entity := range props.Items {
		rows = append(rows, menuframe.Row{Cells: []styled.Cell{styled.IconCell(menuIcon(world, entity)), styled.TextCell(query.GetEntityName(entity, world))}})
	}
	list, pager := menuframe.RenderList(selectedIndex, rows, styled.Cols(styled.Icon(), styled.Name()),
		menuframe.ListOpts{EmptyText: query.T(world, "Nothing to equip")}, res)
	return menuframe.PanelScreen(world, res, query.T(world, "Choose equipment"), list, "", pager)
}

// detailContent は装備選択中の詳細内容を返す。候補なら現装備との差分付き、
// 「外す」なら外す対象の性能を出す。選択が無ければ ok=false
func (o *characterEquipOverlay) detailContent(world w.World) (overlay.DetailContent, bool) {
	choice, ok := o.selection()
	if !ok {
		return overlay.DetailContent{}, false
	}
	props := o.mount.GetProps()
	if choice.unequip {
		if props.PreviousEquipment == nil {
			return overlay.DetailContent{}, false
		}
		return overlay.EntityDetailContent(world, *props.PreviousEquipment), true
	}
	dc := overlay.EntityDetailContent(world, choice.entity)
	dc.Rows = equipCompareRows(world, choice.entity, props.PreviousEquipment)
	return dc, true
}

// equipCompareRows は候補の性能行を返し、現装備 prev と同じ項目には差分を併記する。
// prev が nil や死んだ実体なら差分を付けず候補単体の行を返す。数値として読める項目だけ差分を出し、
// 重量や弾数のような複合値には付けない。有利な変化は緑、不利は赤で塗り、攻撃コストなど
// 小さいほど良い項目は極性を反転する
func equipCompareRows(world w.World, candidate ecs.Entity, prev *ecs.Entity) []entityspec.SpecRow {
	rows := entityspec.SpecRows(world, candidate)
	if prev == nil || !world.ECS.Alive(*prev) {
		return rows
	}
	prevValues := map[string]string{}
	for _, r := range entityspec.SpecRows(world, *prev) {
		if !r.Header {
			prevValues[r.Label] = r.Value
		}
	}
	lowerIsBetter := map[string]bool{
		query.T(world, "Attack cost"): true,
		query.T(world, "Reload"):      true,
	}
	for i := range rows {
		if rows[i].Header {
			continue
		}
		prevVal, found := prevValues[rows[i].Label]
		if !found {
			continue
		}
		candN, err1 := strconv.Atoi(rows[i].Value)
		prevN, err2 := strconv.Atoi(prevVal)
		if err1 != nil || err2 != nil {
			continue
		}
		delta := candN - prevN
		if delta == 0 {
			continue
		}
		rows[i].Value = fmt.Sprintf("%s (%+d)", rows[i].Value, delta)
		better := delta > 0
		if lowerIsBetter[rows[i].Label] {
			better = delta < 0
		}
		rowColor := theme.StatusSuccess
		if !better {
			rowColor = theme.StatusDanger
		}
		rows[i].Color = &rowColor
	}
	return rows
}
