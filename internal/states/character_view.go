package states

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// ViewUI は View の internal/ui 版。人物画面のタブ本体を自前 UI で組む。
// 装備選択の入れ子モーダルは equip overlay が ScreenRenderer として本体の上へ重ねる。
func (st *CharacterState) ViewUI(world w.World, props CharacterProps, cursor menuloop.Selection, res resources.UIResources) ui.Widget {
	var content []ui.Widget
	var pager string
	if charTabAt(cursor.TabIndex) == charTabEquip {
		content, pager = buildEquipListUI(world, props.EquipSlots, cursor.ItemIndex, res)
	} else if infoIdx := cursor.TabIndex - charFirstInfoTab; infoIdx >= 0 && infoIdx < len(props.InfoTabs) {
		content, pager = buildInfoTableUI(world, props.InfoTabs[infoIdx], cursor.ItemIndex, res)
	}
	return menuframe.TabScreen(world, res, props.TargetName, characterTabLabels(world), cursor.TabIndex, content, keybind.HelpHint(world), pager)
}

// buildEquipListUI は buildEquipList の internal/ui 版。装備4列とフッタ右端のページ表示を返す。
func buildEquipListUI(world w.World, slots []equipItemData, itemIndex int, res resources.UIResources) ([]ui.Widget, string) {
	cols := styled.Cols(styled.Name(130), styled.Icon(), styled.Name(140), styled.Num(70))
	rows := make([]menuframe.Row, len(slots))
	for i, slot := range slots {
		var icon *ebiten.Image
		weight := ""
		if slot.Entity != nil {
			icon, _ = resources.SpriteImage(world.Resources.SpriteSheets, world.Components.SpriteRender.Get(*slot.Entity))
			weight = query.GetEntityWeight(world, *slot.Entity).KgString()
		}
		rows[i] = menuframe.Row{Cells: []styled.Cell{styled.TextCell(slot.SlotLabel), styled.IconCell(icon), styled.TextCell(slot.ItemName), styled.TextCell(weight)}}
	}
	return menuframe.RenderList(itemIndex, rows, cols, menuframe.ListOpts{EmptyText: query.T(world, "No equipment slots"), ItemsPerPage: menuframe.ListCapacity(world, true, true)}, res)
}
