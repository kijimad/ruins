package states

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
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
	detail *menuscreen.Detail
}

var _ menuscreen.Overlay = (*characterEquipOverlay)(nil)

// newCharacterEquipOverlay は共有の詳細モーダルを参照する装備選択 overlay を作る
func newCharacterEquipOverlay(detail *menuscreen.Detail) characterEquipOverlay {
	return characterEquipOverlay{mount: hooks.NewMount[charEquipProps](), detail: detail}
}

// Active は装備選択中かを返す
func (o *characterEquipOverlay) Active() bool { return o.active }

// Open は空スロットに対する装備選択を開き、候補を読み込む
func (o *characterEquipOverlay) Open(world w.World, slot equipItemData) {
	o.mount = hooks.NewMount[charEquipProps]()
	o.mount.SetProps(charEquipProps{
		Items:             equipableForSlot(world, slot.SlotNumber),
		SlotNumber:        slot.SlotNumber,
		PreviousEquipment: slot.Entity,
		TargetMember:      slot.Member,
	})
	o.active = true
}

// selectedItem は現在カーソルが指す候補を返す。候補が無ければ ok=false
func (o *characterEquipOverlay) selectedItem() (ecs.Entity, bool) {
	props := o.mount.GetProps()
	ms, _ := hooks.GetState[hooks.TabMenuState](o.mount, "char_equip")
	if ms.ItemIndex < 0 || ms.ItemIndex >= len(props.Items) {
		return gc.InvalidEntity, false
	}
	return props.Items[ms.ItemIndex], true
}

// HandleInput は装備選択中の入力を処理する。毎フレーム呼ばれるので自前カーソル mount の維持もここで行う。
// x で詳細を入れ子に開き、Enter で装着して閉じ、Esc で閉じる
func (o *characterEquipOverlay) HandleInput(world w.World) (bool, error) {
	if !o.active {
		return false, nil
	}
	props := o.mount.GetProps()
	hooks.UseTabMenu(o.mount.Store(), "char_equip", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})

	ki := input.GetSharedKeyboardInput()
	dirty := false
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		o.detail.Open()
		dirty = true
	} else if action, ok := HandleMenuInput(); ok {
		switch action {
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			o.active = false
			dirty = true
		case inputmapper.ActionMenuSelect:
			if err := o.execute(world); err != nil {
				return false, err
			}
			o.active = false
			dirty = true
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
			o.mount.Dispatch(action)
		default:
			// 装備選択中は上記以外のアクションを扱わない
		}
	}
	if o.mount.Update() {
		dirty = true
	}
	return dirty, nil
}

// execute は選んだ候補を装着する。既存の装備があれば持ち物へ戻す
func (o *characterEquipOverlay) execute(world w.World) error {
	props := o.mount.GetProps()
	ms, _ := hooks.GetState[hooks.TabMenuState](o.mount, "char_equip")
	if ms.ItemIndex >= len(props.Items) {
		return nil
	}
	item := props.Items[ms.ItemIndex]
	itemName := query.GetEntityName(item, world)
	if props.PreviousEquipment != nil {
		if err := lifecycle.MoveToBackpack(world, *props.PreviousEquipment, props.TargetMember); err != nil {
			return err
		}
	}
	lifecycle.MoveToEquip(world, item, props.TargetMember, props.SlotNumber)
	logEquipChange(world, props.TargetMember, itemName, "を装備した。")
	return nil
}

// Window は装備候補のサブウィンドウを rect の位置へ組む
func (o *characterEquipOverlay) Window(world w.World, rect image.Rectangle) *widget.Window {
	if !o.active {
		return nil
	}
	props := o.mount.GetProps()
	ms, _ := hooks.GetState[hooks.TabMenuState](o.mount, "char_equip")
	return buildEquipSelectWindow(world, props, ms.ItemIndex, rect, world.Resources.UIResources)
}
