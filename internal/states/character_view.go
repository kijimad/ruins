package states

import (
	"fmt"
	"image"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// View は人物画面のタブ本体を props と選択位置から組み立てる描画。ラベルの訳のみ world から引く。
// 詳細や装備選択のオーバーレイ窓は Screen が重ねる。menurt.Model の View 部にあたる
func (st *CharacterState) View(world w.World, props CharacterProps, cursor menurt.Selection, res resources.UIResources) *ebitenui.UI {
	// 見出しは対象キャラ名。仲間がいれば左右矢印で切替可能を示す。
	// 矢印は素の記号だとフォントに無く文字化けするため FontAwesome のアイコンを使う
	header := props.TargetName
	if props.HasMultiple {
		header = fmt.Sprintf("%s %s %s", consts.IconArrowLeft, props.TargetName, consts.IconArrowRight)
	}

	// コンテンツは現在タブの中身。装備は編集可能、以降は読み取り専用の情報タブ
	var content *widget.Container
	if charTabAt(cursor.TabIndex) == charTabEquip {
		content = buildEquipList(world, props.EquipSlots, cursor.ItemIndex, res)
	} else if charTabAt(cursor.TabIndex) == charTabCommand {
		content = buildCommandTable(world, props.Commands, cursor.ItemIndex, res)
	} else if infoIdx := cursor.TabIndex - charFirstInfoTab; infoIdx >= 0 && infoIdx < len(props.InfoTabs) {
		content = buildInfoTable(world, props.InfoTabs[infoIdx], cursor.ItemIndex, res)
	} else {
		content = widget.NewContainer()
	}

	extras := []string{query.T(world, "x Details")}
	if props.HasMultiple {
		extras = []string{query.T(world, ", . Switch"), query.T(world, "x Details")}
	}

	return newTabScreenUI(res, tabScreen{
		Header:    header,
		TabLabels: characterTabLabels(world),
		TabIndex:  cursor.TabIndex,
		Content:   content,
		Footer:    menuNavHint(world, true, extras...),
	})
}

// buildEquipList は装備タブの一覧を組み立てる。スロット名、装備アイコン、装備名を並べ、
// 未装備はアイコンと名前を空欄にする。アイコンは装備名の左に置く
func buildEquipList(world w.World, slots []equipItemData, itemIndex int, res resources.UIResources) *widget.Container {
	columnWidths := []int{120, itemIconColumnWidth, 220}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignLeft}
	rows := make([]menuRow, len(slots))
	for i, slot := range slots {
		var icon *ebiten.Image
		if slot.Entity != nil {
			icon, _ = resources.SpriteImage(world.Resources.SpriteSheets, world.Components.SpriteRender.Get(*slot.Entity))
		}
		rows[i] = menuRow{Cells: []styled.Cell{styled.TextCell(slot.SlotLabel), styled.IconCell(icon), styled.TextCell(slot.ItemName)}}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No equipment slots")}, res)
}

// buildCommandTable は命令タブの一覧を組み立てる。左に指示名、右に現在の値を並べる
func buildCommandTable(world w.World, cmdRows []commandRow, itemIndex int, res resources.UIResources) *widget.Container {
	columnWidths := []int{180, 160}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft}
	rows := make([]menuRow, len(cmdRows))
	for i, row := range cmdRows {
		rows[i] = menuRow{Cells: styled.TextCells(query.T(world, string(row.Kind)), row.Value)}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No formation orders")}, res)
}

// buildEquipSelectWindow は装備選択のサブウィンドウを rect の位置へ組み立てる。
// 候補名は world から引くため world を受け取るが、選択状態は selectedIndex で明示的に渡す
func buildEquipSelectWindow(world w.World, props charEquipProps, selectedIndex int, rect image.Rectangle, res resources.UIResources) *widget.Window {
	content := styled.NewWindowContainer(res)
	title := styled.NewWindowHeaderContainer(query.T(world, "Choose equipment"), res)
	win := styled.NewSmallWindow(title, content)
	if len(props.Items) == 0 {
		content.AddChild(styled.NewDescriptionText(query.T(world, "Nothing to equip"), res))
	}
	for i, entity := range props.Items {
		name := query.GetEntityName(entity, world)
		icon, _ := resources.SpriteImage(world.Resources.SpriteSheets, world.Components.SpriteRender.Get(entity))
		content.AddChild(styled.NewListItem(icon, name, theme.TextSecondary, i == selectedIndex, res))
	}
	win.SetLocation(rect)
	return win
}
