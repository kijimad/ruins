package states

import (
	"fmt"
	"image"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// buildCharacterUI は人物画面のタブ本体を props と選択位置だけから組み立てる。
// state に触れない純粋描画で、詳細や装備選択のオーバーレイ窓は呼び出し側が重ねる
func buildCharacterUI(props CharacterProps, sel menurt.Selection, res resources.UIResources) *ebitenui.UI {
	// 見出しは対象キャラ名。仲間がいれば左右矢印で切替可能を示す。
	// 矢印は素の記号だとフォントに無く文字化けするため FontAwesome のアイコンを使う
	header := props.TargetName
	if props.HasMultiple {
		header = fmt.Sprintf("%s %s %s", consts.IconArrowLeft, props.TargetName, consts.IconArrowRight)
	}

	// コンテンツは現在タブの中身。装備は編集可能、以降は読み取り専用の情報タブ
	var content *widget.Container
	if charTabAt(sel.TabIndex) == charTabEquip {
		content = buildEquipList(props.EquipSlots, sel.ItemIndex, res)
	} else if charTabAt(sel.TabIndex) == charTabCommand {
		content = buildCommandTable(props.Commands, sel.ItemIndex, res)
	} else if infoIdx := sel.TabIndex - charFirstInfoTab; infoIdx >= 0 && infoIdx < len(props.InfoTabs) {
		content = buildInfoTable(props.InfoTabs[infoIdx], sel.ItemIndex, res)
	} else {
		content = widget.NewContainer()
	}

	extras := []string{"x 詳細"}
	if props.HasMultiple {
		extras = []string{", . 切替", "x 詳細"}
	}

	return newTabScreenUI(res, tabScreen{
		Header:    header,
		TabLabels: characterTabLabels(),
		TabIndex:  sel.TabIndex,
		Content:   content,
		Footer:    menuNavHint(true, extras...),
	})
}

// buildEquipList は装備タブの一覧を組み立てる。左にスロット名、右に装備名を並べ、未装備は空欄にする
func buildEquipList(slots []equipItemData, itemIndex int, res resources.UIResources) *widget.Container {
	columnWidths := []int{120, 220}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft}
	rows := make([]menuRow, len(slots))
	for i, slot := range slots {
		rows[i] = menuRow{Cells: []string{slot.SlotLabel, slot.ItemName}}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: "装備スロットがありません"}, res)
}

// buildCommandTable は命令タブの一覧を組み立てる。左に指示名、右に現在の値を並べる
func buildCommandTable(cmdRows []commandRow, itemIndex int, res resources.UIResources) *widget.Container {
	columnWidths := []int{180, 160}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft}
	rows := make([]menuRow, len(cmdRows))
	for i, row := range cmdRows {
		rows[i] = menuRow{Cells: []string{string(row.Kind), row.Value}}
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: "この対象に隊列指示はない"}, res)
}

// buildEquipSelectWindow は装備選択のサブウィンドウを rect の位置へ組み立てる。
// 候補名は world から引くため world を受け取るが、選択状態は selectedIndex で明示的に渡す
func buildEquipSelectWindow(world w.World, props charEquipProps, selectedIndex int, rect image.Rectangle, res resources.UIResources) *widget.Window {
	content := styled.NewWindowContainer(res)
	title := styled.NewWindowHeaderContainer("装備を選ぶ", res)
	win := styled.NewSmallWindow(title, content)
	if len(props.Items) == 0 {
		content.AddChild(styled.NewDescriptionText("装備できるものがない", res))
	}
	for i, entity := range props.Items {
		name := world.Components.Name.Get(entity).Name
		content.AddChild(styled.NewListItemText(name, theme.TextSecondary, i == selectedIndex, res))
	}
	win.SetLocation(rect)
	return win
}
