package tabmenu

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
)

// UIBuilder はTabMenuのUI要素を構築する
type uiBuilder struct {
	world       w.World
	itemWidgets []widget.PreferredSizeLocateableWidget // 現在表示中のウィジェット
}

// newUIBuilder はUIビルダーを作成する
func newUIBuilder(world w.World) *uiBuilder {
	return &uiBuilder{
		world:       world,
		itemWidgets: make([]widget.PreferredSizeLocateableWidget, 0),
	}
}

// BuildUI はtabMenuのUI要素を構築する（タブが1つの場合を想定）
func (b *uiBuilder) BuildUI(tabMenu *tabMenu) *widget.Container {
	// タブが1つしかない場合は、そのタブのアイテムを直接表示
	// 垂直リスト表示（固定）
	return b.buildVerticalUI(tabMenu)
}

// buildVerticalUI は垂直リスト表示のUIを構築する
func (b *uiBuilder) buildVerticalUI(tabMenu *tabMenu) *widget.Container {
	mainContainer := styled.NewVerticalContainer()
	b.itemWidgets = make([]widget.PreferredSizeLocateableWidget, 0)

	// ページインジケーターを追加
	pageText := tabMenu.GetPageIndicatorText()
	if pageText != "" {
		pageIndicator := b.CreatePageIndicator(tabMenu)
		mainContainer.AddChild(pageIndicator)
	}

	// 表示する項目のみを追加（スクロール対応）
	visibleItems, indices := tabMenu.GetVisibleItems()
	for i, item := range visibleItems {
		originalIndex := indices[i]
		btn := b.CreateMenuButton(tabMenu, originalIndex, item)
		mainContainer.AddChild(btn)
		b.itemWidgets = append(b.itemWidgets, btn)
	}

	b.UpdateFocus(tabMenu)

	return mainContainer
}

// CreateMenuButton はメニューボタンを作成する
func (b *uiBuilder) CreateMenuButton(tabMenu *tabMenu, index int, item Item) widget.PreferredSizeLocateableWidget {
	// フォーカス状態をチェック
	isFocused := index == tabMenu.GetCurrentItemIndex()

	// 無効時は灰色テキスト
	textColor := consts.TextColor
	if item.Disabled {
		textColor = consts.ForegroundColor
	}

	return styled.NewListItemText(
		item.Label,
		textColor,
		isFocused,
		b.world.Resources.UIResources,
		item.AdditionalLabels...,
	)
}

// UpdateFocus はメニューのフォーカス表示を更新する
// カーソルの色を変更して選択状態を表現する
func (b *uiBuilder) UpdateFocus(tabMenu *tabMenu) {
	if len(b.itemWidgets) == 0 {
		return
	}

	// 表示中の項目とそのインデックスを取得
	_, indices := tabMenu.GetVisibleItems()

	// 全てのアイテムのカーソル色を更新
	for i, w := range b.itemWidgets {
		if i >= len(indices) {
			continue
		}

		originalIndex := indices[i]
		isFocused := originalIndex == tabMenu.GetCurrentItemIndex()

		// カーソルの色を決定
		cursorColor := color.RGBA{}
		if isFocused {
			cursorColor = consts.PrimaryColor
		}

		// コンテナの場合、最初の子要素（カーソルText）の色を更新
		if container, ok := w.(*widget.Container); ok {
			children := container.Children()
			if len(children) > 0 {
				if cursorText, ok := children[0].(*widget.Text); ok {
					cursorText.SetColor(cursorColor)
				}
			}
		}
	}
}

// CreatePageIndicator はページインジケーターを作成する
func (b *uiBuilder) CreatePageIndicator(tabMenu *tabMenu) *widget.Text {
	res := b.world.Resources.UIResources

	pageText := tabMenu.GetPageIndicatorText()

	return widget.NewText(
		widget.TextOpts.Text(pageText, &res.Text.SmallFace, color.White),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 20),
		),
	)
}

// UpdateTabDisplayContainer はタブ表示コンテナを更新する
// ページインジケーター、アイテム一覧、空の場合のメッセージを表示する
func (b *uiBuilder) UpdateTabDisplayContainer(container *widget.Container, tabMenu *tabMenu) {
	// 既存の子要素をクリア
	container.RemoveChildren()

	currentTab := tabMenu.GetCurrentTab()
	currentItemIndex := tabMenu.GetCurrentItemIndex()

	// ページインジケーターを表示
	pageText := tabMenu.GetPageIndicatorText()
	if pageText != "" {
		pageIndicator := styled.NewPageIndicator(pageText, b.world.Resources.UIResources)
		container.AddChild(pageIndicator)
	}

	// 現在のページで表示されるアイテムとインデックスを取得
	visibleItems, indices := tabMenu.GetVisibleItems()

	// アイテム一覧を表示（ページ内のアイテムのみ）
	for i, item := range visibleItems {
		actualIndex := indices[i]
		isSelected := actualIndex == currentItemIndex && currentItemIndex >= 0

		// Disabledアイテムの場合は灰色で表示
		if item.Disabled {
			itemWidget := styled.NewListItemText(item.Label, consts.ForegroundColor, isSelected, b.world.Resources.UIResources, item.AdditionalLabels...)
			container.AddChild(itemWidget)
		} else if isSelected {
			// 選択中のアイテムは背景色付きで明るい文字色
			itemWidget := styled.NewListItemText(item.Label, consts.TextColor, true, b.world.Resources.UIResources, item.AdditionalLabels...)
			container.AddChild(itemWidget)
		} else {
			// 非選択のアイテムは背景なしで明るい文字色
			itemWidget := styled.NewListItemText(item.Label, consts.TextColor, false, b.world.Resources.UIResources, item.AdditionalLabels...)
			container.AddChild(itemWidget)
		}
	}

	// アイテムがない場合の表示
	if len(currentTab.Items) == 0 {
		emptyText := styled.NewDescriptionText("(アイテムなし)", b.world.Resources.UIResources)
		container.AddChild(emptyText)
	}
}
