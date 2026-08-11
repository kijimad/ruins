package messagewindow

import (
	eui_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// UIBuilder はTabMenuのUI要素を構築する
type uiBuilder struct {
	world       w.World
	itemWidgets []widget.PreferredSizeLocateableWidget
}

// newUIBuilder はUIビルダーを作成する
func newUIBuilder(world w.World) *uiBuilder {
	return &uiBuilder{
		world:       world,
		itemWidgets: make([]widget.PreferredSizeLocateableWidget, 0),
	}
}

// BuildUI はUI要素を構築する
func (b *uiBuilder) BuildUI(config tabMenuConfig, state ViewState) *widget.Container {
	mainContainer := styled.NewVerticalContainer()
	b.itemWidgets = make([]widget.PreferredSizeLocateableWidget, 0)

	pageText := pageIndicatorText(config, state)
	if pageText != "" {
		pageIndicator := b.createPageIndicator(pageText)
		mainContainer.AddChild(pageIndicator)
	}

	visibleItems, indices := getVisibleItems(config, state)
	for i, item := range visibleItems {
		originalIndex := indices[i]
		isFocused := originalIndex == state.ItemIndex
		btn := b.createMenuButton(item, isFocused)
		mainContainer.AddChild(btn)
		b.itemWidgets = append(b.itemWidgets, btn)
	}

	return mainContainer
}

// createMenuButton はメニューボタンを作成する
func (b *uiBuilder) createMenuButton(item Item, isFocused bool) widget.PreferredSizeLocateableWidget {
	return styled.NewListItem(nil,
		item.Label,
		theme.TextSecondary,
		isFocused,
		b.world.Resources.UIResources,
		item.AdditionalLabels...,
	)
}

// UpdateFocus はメニューのフォーカス表示を更新する
func (b *uiBuilder) UpdateFocus(config tabMenuConfig, state ViewState) {
	if len(b.itemWidgets) == 0 {
		return
	}

	_, indices := getVisibleItems(config, state)

	for i, w := range b.itemWidgets {
		if i >= len(indices) {
			continue
		}

		originalIndex := indices[i]
		isFocused := originalIndex == state.ItemIndex

		wrapper, ok := w.(*widget.Container)
		if !ok {
			continue
		}
		wrapperChildren := wrapper.Children()
		if len(wrapperChildren) == 0 {
			continue
		}
		contentContainer, ok := wrapperChildren[0].(*widget.Container)
		if !ok {
			continue
		}

		if isFocused {
			contentContainer.SetBackgroundImage(b.world.Resources.UIResources.Panel.SelectionBar)
		} else {
			contentContainer.SetBackgroundImage(eui_image.NewNineSliceColor(theme.Transparent))
		}

		textColor := theme.TextSecondary
		if isFocused {
			textColor = theme.TextSelected
		}

		for _, child := range contentContainer.Children() {
			if textWidget, ok := child.(*widget.Text); ok {
				textWidget.SetColor(textColor)
			}
		}
	}
}

// createPageIndicator はページインジケーターを作成する
func (b *uiBuilder) createPageIndicator(pageText string) *widget.Text {
	res := b.world.Resources.UIResources

	return widget.NewText(
		widget.TextOpts.Text(pageText, &res.Text.SmallFace, theme.TextPrimary),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 20),
		),
	)
}
