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
	if charTabAt(cursor.TabIndex) == charTabEquip {
		content = buildEquipListUI(world, props.EquipSlots, cursor.ItemIndex, res)
	} else if infoIdx := cursor.TabIndex - charFirstInfoTab; infoIdx >= 0 && infoIdx < len(props.InfoTabs) {
		content = buildInfoTableUI(world, props.InfoTabs[infoIdx], cursor.ItemIndex, res)
	}
	return buildTabScreenUI(world, res, props.TargetName, characterTabLabels(world), cursor.TabIndex, content, keybind.HelpHint(world))
}

// buildEquipListUI は buildEquipList の internal/ui 版。スロット名・アイコン・装備名・重量の4列を返す。
func buildEquipListUI(world w.World, slots []equipItemData, itemIndex int, res resources.UIResources) []ui.Widget {
	columnWidths := []int{130, itemIconColumnWidth, 140, 70}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignLeft, styled.AlignRight}
	rows := make([]menuRow, len(slots))
	for i, slot := range slots {
		var icon *ebiten.Image
		weight := ""
		if slot.Entity != nil {
			icon, _ = resources.SpriteImage(world.Resources.SpriteSheets, world.Components.SpriteRender.Get(*slot.Entity))
			weight = query.GetEntityWeight(world, *slot.Entity).KgString()
		}
		rows[i] = menuRow{Cells: []styled.Cell{styled.TextCell(slot.SlotLabel), styled.IconCell(icon), styled.TextCell(slot.ItemName), styled.TextCell(weight)}}
	}
	return renderMenuListUI(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No equipment slots"), ItemsPerPage: menuframe.ListCapacity(res, true, true)}, res.Text.BodyFace)
}
